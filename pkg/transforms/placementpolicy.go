/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	mcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
)

// PlacementPolicyResource ...
type PlacementPolicyResource struct {
	node Node
}

// PlacementPolicyResourceBuilder ...
func PlacementPolicyResourceBuilder(p *mcm.PlacementPolicy) *PlacementPolicyResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["replicas"] = int64(0)
	if p.Spec.ClusterReplicas != nil {
		node.Properties["replicas"] = int64(*p.Spec.ClusterReplicas)
	}

	l := len(p.Status.Decisions)
	decisions := make([]string, l)
	for i := 0; i < l; i++ {
		decisions[i] = p.Status.Decisions[i].ClusterName
	}
	node.Properties["decisions"] = decisions

	return &PlacementPolicyResource{node: node}
}

// BuildNode construct the node for the PlacementPolicy Resources
func (p PlacementPolicyResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the PlacementPolicy Resources
func (p PlacementPolicyResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
