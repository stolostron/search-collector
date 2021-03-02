/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformNode(t *testing.T) {
	var n v1.Node
	UnmarshalFile("node.json", &n, t)
	node := NodeResourceBuilder(&n).BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("architecture", node.Properties["architecture"], "amd64", t)
	AssertEqual("cpu", node.Properties["cpu"], int64(8), t)
	AssertEqual("osImage", node.Properties["osImage"], "Ubuntu 16.04.5 LTS", t)
	AssertEqual("_systemUUID", node.Properties["_systemUUID"], "4BCDE0D7-CFFB-4A8F-B6F8-0026F347AD93", t)
	AssertDeepEqual("role", node.Properties["role"], []string{"etcd", "management", "master", "proxy", "va"}, t)
}

func TestNodeBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource node.json
	var n v1.Node
	UnmarshalFile("node.json", &n, t)
	edges := NodeResourceBuilder(&n).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Node has no edges:", len(edges), 0, t)
}
