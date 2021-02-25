/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// DeploymentResource ...
type DeploymentResource struct {
	node Node
}

// DeploymentResourceBuilder ...
func DeploymentResourceBuilder(d *v1.Deployment) *DeploymentResource {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["available"] = int64(d.Status.AvailableReplicas)
	node.Properties["current"] = int64(d.Status.Replicas)
	node.Properties["ready"] = int64(d.Status.ReadyReplicas)
	node.Properties["desired"] = int64(0)
	if d.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*d.Spec.Replicas)
	}

	return &DeploymentResource{node: node}
}

// BuildNode construct the node for the Deployment Resources
func (d DeploymentResource) BuildNode() Node {
	return d.node
}

// BuildEdges construct the edges for the Deployment Resources
func (d DeploymentResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
