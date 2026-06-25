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
	app "github.com/stolostron/multicloud-operators-placementrule/pkg/apis/apps/v1"
)

// PlacementRuleResource ...
type PlacementRuleResource struct {
	node Node
}

// PlacementRuleResourceBuilder ...
func PlacementRuleResourceBuilder(p *app.PlacementRule) *PlacementRuleResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Add replicas property
	if p.Spec.ClusterReplicas != nil {
		node.Properties["replicas"] = int32(*p.Spec.ClusterReplicas)
	}

	return &PlacementRuleResource{node: node}
}

// BuildNode construct the node for the PlacementRule Resources
func (p PlacementRuleResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the PlacementRule Resources
func (p PlacementRuleResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
