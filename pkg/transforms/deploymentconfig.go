/*
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	v1 "github.com/openshift/api/apps/v1"
)

// DeploymentConfigResource ...
type DeploymentConfigResource struct {
	node Node
}

// DeploymentConfigResourceBuilder ...
func DeploymentConfigResourceBuilder(d *v1.DeploymentConfig) *DeploymentConfigResource {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["available"] = int64(d.Status.AvailableReplicas)
	node.Properties["current"] = int64(d.Status.Replicas)
	node.Properties["ready"] = int64(d.Status.ReadyReplicas)
	node.Properties["desired"] = int64(d.Spec.Replicas)

	return &DeploymentConfigResource{node: node}
}

// BuildNode construct the node for the Deployment Resources
func (d DeploymentConfigResource) BuildNode() Node {
	return d.node
}

// BuildEdges construct the edges for the Deployment Resources
func (d DeploymentConfigResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
