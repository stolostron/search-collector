/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020, 2021 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NodeResource ...
type NodeResource struct {
	node Node
}

// NodeResourceBuilder ...
func NodeResourceBuilder(n *v1.Node, r *unstructured.Unstructured, additionalColumns ...ExtractProperty) *NodeResource {
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

	// sort in alphabetical order to make the ui consistent
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

	// Status logic is based on
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/printers.go#L1765
	// Status must be a string to avoid issues with other resources that have a status field.
	status := "Unknown"
	for _, condition := range n.Status.Conditions {
		if condition.Type == v1.NodeReady {
			if condition.Status == v1.ConditionTrue {
				status = string(condition.Type)
			} else {
				status = "Not" + string(condition.Type)
			}
		}
	}
	if n.Spec.Unschedulable {
		status += "-SchedulingDisabled" // Encoding to single string to work around limitations.
	}
	node.Properties["status"] = status

	node = applyDefaultTransformConfig(node, r, additionalColumns...)

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
