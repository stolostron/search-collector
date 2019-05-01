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

func HelmTransformation(helmClient *helm.Client, output chan NodeEvent) {
	for {
		glog.Info("Fetching helm releases")

		releases, err := helmClient.ListReleases()

		if err != nil {
			glog.Error("Failed to fetch helm releases.  Original error: ", err)
		} else {
			for _, release := range releases.Releases {
				upsert := NodeEvent{
					Time:      time.Now().Unix(),
					Operation: Create,
					Node:      transformRelease(release),
				}
				output <- upsert
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func transformRelease(resource *release.Release) Node {
	lastDeployed := resource.GetInfo().GetLastDeployed()
	timestamp, err := ptypes.Timestamp(lastDeployed)
	if err != nil {
		glog.Errorf("Error coneverting %v to native timestamp in helm transform", lastDeployed)
	}

	node := Node{
		UID:        "Release/" + resource.GetName(),
		Properties: make(map[string]interface{}),
	}
	node.Properties["kind"] = "Release"
	node.Properties["chartName"] = resource.GetChart().GetMetadata().GetName()
	node.Properties["chartVersion"] = resource.GetChart().GetMetadata().GetVersion()
	node.Properties["namespace"] = resource.GetNamespace()
	node.Properties["status"] = release.Status_Code_name[int32(resource.GetInfo().GetStatus().GetCode())]
	node.Properties["revision"] = resource.GetVersion()
	node.Properties["name"] = resource.GetName()
	node.Properties["updated"] = timestamp.UTC().Format(time.RFC3339)
	if config.Cfg.DeployedInHub {
		node.Properties["_hubClusterResource"] = true
	} else {
		node.Properties["_clusterNamespace"] = config.Cfg.ClusterNamespace
	}

	return node
}
