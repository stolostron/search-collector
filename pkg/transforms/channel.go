/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	app "github.com/open-cluster-management/multicloud-operators-channel/pkg/apis/apps/v1"
)

type ChannelResource struct {
	*app.Channel
}

func (c ChannelResource) BuildNode() Node {
	node := transformCommon(c)
	apiGroupVersion(c.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["type"] = string(c.Spec.Type)
	node.Properties["pathname"] = c.Spec.Pathname

	return node
}

// See documentation at pkg/transforms/README.md
func (c ChannelResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(c.UID)

	nodeInfo := NodeInfo{NameSpace: c.Namespace, UID: UID, EdgeType: "uses", Kind: c.Kind, Name: c.Name}

	//Build uses edges to connect channel to configmaps and secrets
	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})
	if c.Spec.ConfigMapRef != nil {
		configmapMap[c.Spec.ConfigMapRef.Name] = struct{}{}
	}
	if c.Spec.SecretRef != nil {
		secretMap[c.Spec.SecretRef.Name] = struct{}{}
	}

	ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns)...)

	//deploys edges
	//HelmRepo channel to deployables edges
	if c.Spec.Type == "HelmRepo" {
		deployables := ns.ByKindNamespaceName["Deployable"][c.Namespace]
		if len(deployables) > 1 {
			nodeInfo.EdgeType = "deploys"
			deployableMap := make(map[string]struct{}, len(deployables))
			for deployable := range deployables {
				deployableMap[deployable] = struct{}{}
			}
			ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns)...)
		}
	}
	return ret
}
