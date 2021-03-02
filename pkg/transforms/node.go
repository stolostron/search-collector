/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// NodeResource ...
type NodeResource struct {
	node Node
}

// NodeResourceBuilder ...
func NodeResourceBuilder(n *v1.Node) *NodeResource {
	node := transformCommon(n) // Start off with the common properties

	var roles []string
	labels := n.ObjectMeta.Labels
	for key, value := range labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") && value == "" {
			roles = append(roles, strings.TrimPrefix(key, "node-role.kubernetes.io/"))
		}
	}

	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	// sort in alphabetical order to make the ui consistant
	sort.Strings(roles)

	apiGroupVersion(n.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["architecture"] = n.Status.NodeInfo.Architecture
	node.Properties["cpu"], _ = n.Status.Capacity.Cpu().AsInt64()
	node.Properties["osImage"] = n.Status.NodeInfo.OSImage
	// Workaround a bug in cAdvisor on ppc64le (see https://github.com/google/cadvisor/pull/2811)
	// that causes a trailing null character in SystemUUID.
	node.Properties["_systemUUID"] = strings.TrimRight(n.Status.NodeInfo.SystemUUID, "\000")
	node.Properties["role"] = roles

	return &NodeResource{node: node}
}

// BuildNode construct the node for the Node Resources
func (n NodeResource) BuildNode() Node {
	return n.node
}

// BuildEdges construct the edges for the Node Resources
func (n NodeResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
