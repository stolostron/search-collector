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
	mcm "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

// DeployableResource ...
type DeployableResource struct {
	node Node
}

// DeployableResourceBuilder ...
func DeployableResourceBuilder(d *mcm.Deployable) *DeployableResource {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type

	node.Properties["chartUrl"] = ""
	node.Properties["deployerNamespace"] = ""

	return &DeployableResource{node: node}
}

// BuildNode construct the node for the Deployable Resources
func (d DeployableResource) BuildNode() Node {
	return d.node
}

// BuildEdges construct the edges for the Deployable Resources
func (d DeployableResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
