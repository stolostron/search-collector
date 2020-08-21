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
	roleSet := map[string]struct{}{
		"node-role.kubernetes.io/proxy":      {},
		"node-role.kubernetes.io/management": {},
		"node-role.kubernetes.io/master":     {},
		"node-role.kubernetes.io/va":         {},
		"node-role.kubernetes.io/etcd":       {},
		"node-role.kubernetes.io/worker":     {},
	}

	for key, value := range labels {
		if _, found := roleSet[key]; found && value == "" {
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
