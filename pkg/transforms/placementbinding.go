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
	"fmt"

	policy "github.com/stolostron/governance-policy-propagator/pkg/apis/policy/v1"
)

// PlacementBindingResource ...
type PlacementBindingResource struct {
	node Node
}

// PlacementBindingResourceBuilder ...
func PlacementBindingResourceBuilder(p *policy.PlacementBinding) *PlacementBindingResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	name := p.PlacementRef.Name
	kind := p.PlacementRef.Kind
	node.Properties["placementpolicy"] = fmt.Sprintf("%s (%s)", name, kind)

	l := len(p.Subjects)
	subjects := make([]string, l)
	for i := 0; i < l; i++ {
		name := p.Subjects[i].Name
		kind := p.Subjects[i].Kind
		subjects[i] = fmt.Sprintf("%s (%s)", name, kind)
	}
	node.Properties["subject"] = subjects

	return &PlacementBindingResource{node: node}
}

// BuildNode construct the node for the PlacementBindingResource Resources
func (p PlacementBindingResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the PlacementBindingResource Resources
func (p PlacementBindingResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
