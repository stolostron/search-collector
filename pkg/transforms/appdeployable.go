/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	appDeployable "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

// AppDeployableResource ...
type AppDeployableResource struct {
	node Node
	Spec appDeployable.DeployableSpec
}

// AppDeployableResourceBuilder ...
func AppDeployableResourceBuilder(d *appDeployable.Deployable) *AppDeployableResource {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	//TODO: Add properties, TEMPLATE-KIND   TEMPLATE-APIVERSION    AGE   STATUS
	if d.Status.Phase != "" {
		node.Properties["status"] = d.Status.Phase
	}
	return &AppDeployableResource{node: node, Spec: d.Spec}
}

// BuildNode construct the node for the AppDeployable Resources
func (d AppDeployableResource) BuildNode() Node {
	return d.node
}

// BuildEdges construct the edges for the AppDeployable Resources
// See documentation at pkg/transforms/README.md
func (d AppDeployableResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := d.node.UID

	nodeInfo := NodeInfo{
		NameSpace: d.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "promotedTo",
		Kind:      d.node.Properties["kind"].(string),
		Name:      d.node.Properties["name"].(string)}

	channelMap := make(map[string]struct{})
	// promotedTo edges
	if d.Spec.Channels != nil {
		for _, channel := range d.Spec.Channels {
			channelMap[channel] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(channelMap, "Channel", nodeInfo, ns, []string{})...)
	}

	// refersTo edges
	// Builds edges between deployable and placement rule
	if d.Spec.Placement != nil && d.Spec.Placement.PlacementRef != nil && d.Spec.Placement.PlacementRef.Name != "" {
		nodeInfo.EdgeType = "refersTo"
		placementRuleMap := make(map[string]struct{})
		placementRuleMap[d.Spec.Placement.PlacementRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementRuleMap, "PlacementRule", nodeInfo, ns, []string{})...)
	}
	return ret
}
