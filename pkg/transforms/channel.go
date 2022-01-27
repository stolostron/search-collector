/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	app "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
)

// ChannelResource ...
type ChannelResource struct {
	node Node
	Spec app.ChannelSpec
}

// ChannelResourceBuilder ...
func ChannelResourceBuilder(c *app.Channel) *ChannelResource {
	node := transformCommon(c)
	apiGroupVersion(c.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["type"] = string(c.Spec.Type)
	node.Properties["pathname"] = c.Spec.Pathname

	// Need to pass spec so we can access it when building the edges.
	return &ChannelResource{node: node, Spec: c.Spec}
}

// BuildNode construct the node for the Channel Resources
func (c ChannelResource) BuildNode() Node {
	return c.node
}

// BuildEdges construct the edges for the Channel Resources
// See documentation at pkg/transforms/README.md
func (c ChannelResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := c.node.UID

	nodeInfo := NodeInfo{
		NameSpace: c.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "uses",
		Kind:      c.node.Properties["kind"].(string),
		Name:      c.node.Properties["name"].(string)}

	// Build uses edges to connect channel to configmaps and secrets
	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})

	if c.Spec.ConfigMapRef != nil {
		configmapMap[c.Spec.ConfigMapRef.Name] = struct{}{}
	}
	if c.Spec.SecretRef != nil {
		secretMap[c.Spec.SecretRef.Name] = struct{}{}
	}

	ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns, []string{})...)
	ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns, []string{})...)

	// deploys edges
	// HelmRepo channel to deployables edges
	if c.Spec.Type == "HelmRepo" {
		deployables := ns.ByKindNamespaceName["Deployable"][c.node.Properties["namespace"].(string)]
		if len(deployables) > 1 {
			nodeInfo.EdgeType = "deploys"
			deployableMap := make(map[string]struct{}, len(deployables))
			for deployable := range deployables {
				deployableMap[deployable] = struct{}{}
			}
			ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns, []string{})...)
		}
	}
	return ret
}
