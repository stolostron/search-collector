/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type HelmResource struct {
	rel *release.Release
}

// Helm Releases are outside the normal k8s paradigm, so this is a separate transformer routine to the one in transformer.go
func HelmTransformation(helmClient helm.Interface, ticker <-chan time.Time, output chan NodeEvent) {
	allStatuses := helm.ReleaseListStatuses([]release.Status_Code{
		release.Status_UNKNOWN,
		release.Status_DEPLOYED,
		release.Status_DELETED,
		release.Status_SUPERSEDED,
		release.Status_FAILED,
		release.Status_DELETING,
		release.Status_PENDING_INSTALL,
		release.Status_PENDING_UPGRADE,
		release.Status_PENDING_ROLLBACK,
	})
	knownReleases := make(map[string]struct{})

	for {
		glog.V(2).Info("Fetching helm releases")

		releases, err := helmClient.ListReleases(allStatuses)

		if err != nil {
			glog.Error("Failed to fetch helm releases. Original error: ", err)
		} else {
			currentReleases := make(map[string]struct{})

			for _, release := range releases.Releases {
				hr := HelmResource{
					rel: release,
				}
				upsert := NodeEvent{
					Time:         time.Now().Unix(),
					Operation:    Create,
					Node:         hr.BuildNode(),
					ComputeEdges: hr.BuildEdges,
				}
				output <- upsert
				currentReleases[upsert.Node.UID] = struct{}{}
			}

			// compute if we need delete events
			for uid := range knownReleases {
				if _, ok := currentReleases[uid]; !ok {
					deleteOp := NodeEvent{
						Time:      time.Now().Unix(),
						Operation: Delete,
						Node: Node{
							UID: uid,
						},
					}
					output <- deleteOp
					delete(knownReleases, uid)
				}
			}

			// save the previous state
			knownReleases = currentReleases
		}

		<-ticker // wait until we get the next tick from our timer
	}
}

func (h HelmResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}

func (h HelmResource) BuildNode() Node {
	lastDeployed := h.rel.GetInfo().GetLastDeployed()
	timestamp, err := ptypes.Timestamp(lastDeployed)
	if err != nil {
		glog.Errorf("Error converting %v to native timestamp in helm transform", lastDeployed)
	}

	node := Node{
		UID:        config.Cfg.ClusterName + "/Release/" + h.rel.GetName(),
		Properties: make(map[string]interface{}),
	}

	node.ResourceString = "releases"

	node.Properties["kind"] = "Release"
	node.Properties["chartName"] = h.rel.GetChart().GetMetadata().GetName()
	node.Properties["chartVersion"] = h.rel.GetChart().GetMetadata().GetVersion()
	node.Properties["namespace"] = h.rel.GetNamespace()
	node.Properties["status"] = release.Status_Code_name[int32(h.rel.GetInfo().GetStatus().GetCode())]
	node.Properties["revision"] = h.rel.GetVersion()
	node.Properties["name"] = h.rel.GetName()
	node.Properties["updated"] = timestamp.UTC().Format(time.RFC3339)
	if config.Cfg.DeployedInHub {
		node.Properties["_hubClusterResource"] = true
	} else {
		node.Properties["_clusterNamespace"] = config.Cfg.ClusterNamespace
	}

	return node
}
