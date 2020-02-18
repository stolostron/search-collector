/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	app "github.com/IBM/multicloud-operators-subscription-release/pkg/apis/app/v1alpha1"
)

type AppHelmCRResource struct {
	*app.HelmRelease
}

func (a AppHelmCRResource) BuildNode() Node {
	node := transformCommon(a)         // Start off with the common properties
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version
	// Add other properties
	if a.Spec.Source != nil && a.Spec.Source.SourceType != "" {
		node.Properties["sourceType"] = a.Spec.Source.SourceType
		if a.Spec.Source.SourceType == "GitHub" {
			node.Properties["url"] = a.Spec.Source.GitHub.Urls
			node.Properties["chartPath"] = a.Spec.Source.GitHub.ChartPath
			node.Properties["branch"] = a.Spec.Source.GitHub.Branch
		} else if a.Spec.Source.SourceType == "HelmRepo" {
			node.Properties["url"] = a.Spec.Source.HelmRepo.Urls
		}
	}
	return node
}

func (a AppHelmCRResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(a.UID)

	nodeInfo := NodeInfo{NameSpace: a.Namespace, UID: UID, Kind: a.Kind, Name: a.Name, EdgeType: "attachedTo"}

	//attachedTo edges
	releaseMap := make(map[string]struct{})

	if a.ObjectMeta.Name != "" {
		releaseMap[a.ObjectMeta.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(releaseMap, "Release", nodeInfo, ns)...)
	}
	if a.Spec.SecretRef != nil {
		secretMap := make(map[string]struct{})
		if a.Spec.SecretRef.Name != "" {
			secretMap[a.Spec.SecretRef.Name] = struct{}{}
			ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns)...)
		}
	}
	if a.Spec.ConfigMapRef != nil {
		configmapMap := make(map[string]struct{})
		if a.Spec.ConfigMapRef.Name != "" {
			configmapMap[a.Spec.ConfigMapRef.Name] = struct{}{}
			ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns)...)
		}
	}
	return ret
}
