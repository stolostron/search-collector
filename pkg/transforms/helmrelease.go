/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type HelmReleaseResource struct {
	*v1.ConfigMap    // ConfigMap Resource that contains information concerning HelmRelease
	*release.Release // Release from Tiller
}

func (h HelmReleaseResource) BuildNode() Node {
	releaseLabels := h.GetLabels()
	releaseName := releaseLabels["NAME"]

	node := Node{
		UID:        config.Cfg.ClusterName + "/Release/" + releaseName,
		Properties: make(map[string]interface{}),
	}
	node.Properties["kind"] = "Release"
	node.Properties["name"] = releaseName
	node.Properties["status"] = releaseLabels["STATUS"]
	revision, err := strconv.ParseInt(releaseLabels["VERSION"], 0, 64)
	if err != nil {
		node.Properties["revision"] = releaseLabels["VERSION"]
	} else {
		node.Properties["revision"] = revision
	}

	if config.Cfg.DeployedInHub {
		node.Properties["_hubClusterResource"] = true
	} else {
		node.Properties["_clusterNamespace"] = config.Cfg.ClusterNamespace
	}

	if h.Release == nil { // if release retrieval from Tiller failed...
		return node // return node with partially-filled Properties
	}

	lastDeployed := h.Release.GetInfo().GetLastDeployed()
	timestamp, _ := ptypes.Timestamp(lastDeployed)
	node.Properties["chartName"] = h.Release.GetChart().GetMetadata().GetName()
	node.Properties["chartVersion"] = h.Release.GetChart().GetMetadata().GetVersion()
	node.Properties["namespace"] = h.Release.GetNamespace()
	node.Properties["updated"] = timestamp.UTC().Format(time.RFC3339)

	return node
}

func (h HelmReleaseResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
