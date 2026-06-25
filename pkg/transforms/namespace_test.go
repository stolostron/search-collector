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
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformNamespace(t *testing.T) {
	var n v1.Namespace
	UnmarshalFile("namespace.json", &n, t)
	node := NamespaceResourceBuilder(&n).BuildNode()

	// Test only the fields that exist in namespace - the common test will test the other bits
	AssertEqual("status", node.Properties["status"], "Active", t)
}

func TestNamespaceBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource namespace.json
	var ns v1.Namespace
	UnmarshalFile("namespace.json", &ns, t)
	edges := NamespaceResourceBuilder(&ns).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Namespace has no edges:", len(edges), 0, t)
}
