/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// JobResource ...
type JobResource struct {
	node Node
}

// JobResourceBuilder ...
func JobResourceBuilder(j *v1.Job, r *unstructured.Unstructured) *JobResource {
	node := transformCommon(j)         // Start off with the common properties
	apiGroupVersion(j.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["successful"] = int64(j.Status.Succeeded)
	node.Properties["completions"] = int64(0)
	if j.Spec.Completions != nil {
		node.Properties["completions"] = int64(*j.Spec.Completions)
	}
	node.Properties["parallelism"] = int64(0)
	if j.Spec.Completions != nil {
		node.Properties["parallelism"] = int64(*j.Spec.Parallelism)
	}

	node = applyDefaultTransformConfig(node, r)

	return &JobResource{node: node}
}

// BuildNode construct node for Job resources
func (j JobResource) BuildNode() Node {
	return j.node
}

// BuildEdges construct edges for Job resources
func (j JobResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
