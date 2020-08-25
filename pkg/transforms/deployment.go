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
