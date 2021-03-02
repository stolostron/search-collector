/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// DaemonSetResource ...
type DaemonSetResource struct {
	node Node
}

// DaemonSetResourceBuilder ...
func DaemonSetResourceBuilder(d *v1.DaemonSet) *DaemonSetResource {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["available"] = int64(d.Status.NumberAvailable)
	node.Properties["current"] = int64(d.Status.CurrentNumberScheduled)
	node.Properties["desired"] = int64(d.Status.DesiredNumberScheduled)
	node.Properties["ready"] = int64(d.Status.NumberReady)
	node.Properties["updated"] = int64(d.Status.UpdatedNumberScheduled)

	return &DaemonSetResource{node: node}
}

// BuildNode construct the node for the Daemonset Resources
func (d DaemonSetResource) BuildNode() Node {
	return d.node
}

// BuildEdges construct the edges for the Daemonset Resources
func (d DaemonSetResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
