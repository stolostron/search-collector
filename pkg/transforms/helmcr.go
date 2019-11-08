/*
IBM Confidential
OCO Source Materials
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
	node := transformCommon(h)         // Start off with the common properties
	apiGroupVersion(h.TypeMeta, &node) // add kind, apigroup and version
	//TODO: Add other properties, if any
	return node
}

func (h HelmCRResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(h.UID)

	nodeInfo := NodeInfo{NameSpace: h.Namespace, UID: UID, Kind: h.Kind, Name: h.Name, EdgeType: "attachedTo"}

	//attachedTo edges
	releaseMap := make(map[string]struct{})
	// Connect to Helm Release
	if h.Spec.ReleaseName != "" {
		destUID := GetHelmReleaseUID(h.Spec.ReleaseName)
		releaseMap[h.Spec.ReleaseName] = struct{}{}
		// Propagate hosting Subscription/Deployable properties from the helmCR to helm release so that we can track helm release's deployments and connect them back to the subscription/application
		releaseNode := ns.ByUID[destUID]
		crNode := ns.ByUID[UID]
		//Copy the properties only if the node doesn't have it yet or if they are not the same
		if _, ok := releaseNode.Properties["_hostingSubscription"]; !ok && crNode.Properties["_hostingSubscription"] != releaseNode.Properties["_hostingSubscription"] {
			copyhostingSubProperties(UID, destUID, ns)
		}
		ret = append(ret, edgesByDestinationName(releaseMap, "Release", nodeInfo, ns)...)
	}
	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
