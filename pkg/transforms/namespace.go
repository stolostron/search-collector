/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// NamespaceResource ...
type NamespaceResource struct {
	node Node
}

// NamespaceResourceBuilder ...
func NamespaceResourceBuilder(n *v1.Namespace) *NamespaceResource {
	node := transformCommon(n)         // Start off with the common properties
	apiGroupVersion(n.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["status"] = string(n.Status.Phase)

	return &NamespaceResource{node: node}
}

// BuildNode construct the node for the Namespace Resources
func (n NamespaceResource) BuildNode() Node {
	return n.node
}

// BuildEdges construct the edges for the Namespace Resources
func (n NamespaceResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
