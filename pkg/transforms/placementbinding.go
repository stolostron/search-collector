/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"fmt"

	mcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
)

// PlacementBindingResource ...
type PlacementBindingResource struct {
	node               Node
	PlacementPolicyRef mcm.PlacementPolicyRef
}

// PlacementBindingResourceBuilder ...
func PlacementBindingResourceBuilder(p *mcm.PlacementBinding) *PlacementBindingResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	name := p.PlacementPolicyRef.Name
	kind := p.PlacementPolicyRef.Kind
	node.Properties["placementpolicy"] = fmt.Sprintf("%s (%s)", name, kind)

	l := len(p.Subjects)
	subjects := make([]string, l)
	for i := 0; i < l; i++ {
		name := p.Subjects[i].Name
		kind := p.Subjects[i].Kind
		subjects[i] = fmt.Sprintf("%s (%s)", name, kind)
	}
	node.Properties["subject"] = subjects

	return &PlacementBindingResource{node: node, PlacementPolicyRef: p.PlacementPolicyRef}
}

// BuildNode construct the node for the PlacementBindingResource Resources
func (p PlacementBindingResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the PlacementBindingResource Resources
func (p PlacementBindingResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := p.node.UID

	// refersTo edges
	// Builds edges between placement binding and placement policy.

	nodeInfo := NodeInfo{
		NameSpace: p.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "refersTo",
		Kind:      p.node.Properties["kind"].(string),
		Name:      p.node.Properties["name"].(string)}

	if p.PlacementPolicyRef.Name != "" {
		placementPolicyMap := make(map[string]struct{})
		placementPolicyMap[p.PlacementPolicyRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementPolicyMap, "PlacementPolicy", nodeInfo, ns, []string{})...)
	}
	return ret
}
