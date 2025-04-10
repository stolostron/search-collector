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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformNode(t *testing.T) {
	var n v1.Node
	UnmarshalFile("node.json", &n, t)
	node := NodeResourceBuilder(&n, newUnstructuredNode()).BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("architecture", node.Properties["architecture"], "amd64", t)
	AssertEqual("cpu", node.Properties["cpu"], int64(8), t)
	AssertEqual("osImage", node.Properties["osImage"], "Ubuntu 16.04.5 LTS", t)
	AssertEqual("_systemUUID", node.Properties["_systemUUID"], "4BCDE0D7-CFFB-4A8F-B6F8-0026F347AD93", t)
	AssertDeepEqual("role", node.Properties["role"], []string{"etcd", "main", "management", "proxy", "va"}, t)
	AssertDeepEqual("status", node.Properties["status"], "Ready", t)
	AssertEqual("ipAddress", node.Properties["ipAddress"], "1.1.1.1", t)
	AssertEqual("memoryCapacity", node.Properties["memoryCapacity"], int64(25281953792), t)       // 24689408Ki * 1024
	AssertEqual("memoryAllocatable", node.Properties["memoryAllocatable"], int64(24103354368), t) // 23538432Ki * 1024
}

func TestNodeBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource node.json
	var n v1.Node
	UnmarshalFile("node.json", &n, t)
	edges := NodeResourceBuilder(&n, newUnstructuredNode()).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Node has no edges:", len(edges), 0, t)
}

func newUnstructuredNode() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Node",
		"status": map[string]interface{}{
			"addresses": []interface{}{
				map[string]interface{}{
					"address": "1.1.1.1",
					"type":    "InternalIP",
				},
				map[string]interface{}{
					"address": "1.1.1.1",
					"type":    "Hostname",
				},
			},
			"allocatable": map[string]interface{}{
				"cpu":               "7600m",
				"ephemeral-storage": "240844852Ki",
				"hugepages-1Gi":     "0",
				"hugepages-2Mi":     "0",
				"memory":            "23538432Ki",
				"pods":              "80",
			},
			"capacity": map[string]interface{}{
				"cpu":               "8",
				"ephemeral-storage": "243044404Ki",
				"hugepages-1Gi":     "0",
				"hugepages-2Mi":     "0",
				"memory":            "24689408Ki",
				"pods":              "80",
			},
		},
	}}
}
