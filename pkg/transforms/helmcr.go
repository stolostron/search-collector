/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	app "github.ibm.com/IBMMulticloudPlatform/helm-crd/pkg/apis/helm.bitnami.com/v1"
)

type HelmCRResource struct {
	*app.HelmRelease
}

func (h HelmCRResource) BuildNode() Node {
	node := transformCommon(h) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "HelmRelease"
	node.Properties["apigroup"] = "app.ibm.com"
	//TODO: Add other properties, if any
	return node
}

func (h HelmCRResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(h.UID)

	nodeInfo := NodeInfo{NameSpace: h.Namespace, UID: UID, Kind: h.Kind, Name: h.Name, EdgeType: "attachedTo"}

	//attachedTo edges
	releaseMap := make(map[string]struct{})

	if h.Spec.ReleaseName != "" {
		releaseMap[h.Spec.ReleaseName] = struct{}{}
		ret = append(ret, edgesByDestinationName(releaseMap, "Release", nodeInfo, ns)...)
	}
	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
