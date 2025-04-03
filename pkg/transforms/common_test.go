/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
*/

package transforms

import (
	"testing"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	testLabels = map[string]string{"app": "test", "fake": "true", "component": "testapp"}
	timestamp  = machineryV1.Now()
)

// Helper function for creating a k8s resource to pass in to tests.
// In this case it's a pod.
func CreateGenericResource() machineryV1.Object {
	// Construct an object to test with, in this case a Pod with some of its fields blank.
	p := v1.Pod{}
	p.APIVersion = "v1"
	p.Name = "testpod"
	p.Namespace = "default"
	p.UID = "00aa0000-00aa-00a0-a000-00000a00a0a0"
	p.CreationTimestamp = timestamp
	p.Labels = testLabels
	p.Annotations = map[string]string{"hello": "world"}
	return &p
}

func TestCommonProperties(t *testing.T) {
	config.Cfg.CollectAnnotations = true

	defer func() {
		config.Cfg.CollectAnnotations = false
	}()

	res := CreateGenericResource()
	timeString := timestamp.UTC().Format(time.RFC3339)

	cp := commonProperties(res)

	// Test all the fields.
	AssertEqual("name", cp["name"], interface{}("testpod"), t)
	AssertEqual("namespace", cp["namespace"], interface{}("default"), t)
	AssertEqual("created", cp["created"], interface{}(timeString), t)
	AssertEqual("annotation", cp["annotation"].(map[string]string)["hello"], "world", t)

	noLabels := true
	for key, value := range cp["label"].(map[string]string) {
		noLabels = false
		if testLabels[key] != value {
			t.Error("Incorrect label: " + key)
			t.Fail()
		}
	}

	if noLabels {
		t.Error("No labels found on resource")
		t.Fail()
	}
}

func TestKyvernoPolicyEdges(t *testing.T) {
	configmap := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "zk-kafka-address",
			"uid":       "18b016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test2",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "zk-kafka-address",
				"generate.kyverno.io/policy-namespace":  "",
				"generate.kyverno.io/rule-name":         "k-kafka-address",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Namespace",
				"generate.kyverno.io/trigger-namespace": "",
				"generate.kyverno.io/trigger-uid":       "12345",
				"generate.kyverno.io/trigger-version":   "v1",
				"somekey":                               "somevalue",
			},
		},
	}}

	configmapTwo := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "hello-cofigmap",
			"uid":       "777016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test2",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "kyverno-policy-test",
				"generate.kyverno.io/policy-namespace":  "test2",
				"generate.kyverno.io/rule-name":         "k-kafka-address",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Namespace",
				"generate.kyverno.io/trigger-namespace": "",
			},
		},
	}}

	kyvernoPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "Policy",
			"metadata": map[string]interface{}{
				"name":      "kyverno-policy-test",
				"namespace": "test2",
				"uid":       "777738fa-0591-44da-8a06-98ea7d74a7f7",
			},
		},
	}

	kyvernoClusterpolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "zk-kafka-address",
				"uid":  "8fc338fa-0591-44da-8a06-98ea7d74a7f7",
			},
		},
	}

	nodes := []Node{
		KyvernoPolicyResourceBuilder(kyvernoPolicy).BuildNode(),
		KyvernoPolicyResourceBuilder(kyvernoClusterpolicy).BuildNode(),
		GenericResourceBuilder(configmap).BuildNode(),
		GenericResourceBuilder(configmapTwo).BuildNode(),
	}
	nodeStore := BuildFakeNodeStore(nodes)

	t.Log("Test Kyverno Cluster Policy")
	edges := CommonEdges("local-cluster/18b016fe-1931-4e80-95d1-d51d3b936e24", nodeStore)
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge but got %d", len(edges))
	}

	edge := edges[0]
	expectedEdge := Edge{
		EdgeType:   "generatedBy",
		SourceUID:  "local-cluster/18b016fe-1931-4e80-95d1-d51d3b936e24",
		DestUID:    "local-cluster/8fc338fa-0591-44da-8a06-98ea7d74a7f7",
		SourceKind: "ConfigMap",
		DestKind:   "ClusterPolicy",
	}

	AssertDeepEqual("edge", edge, expectedEdge, t)

	t.Log("Test Kyverno Policy")
	edges = CommonEdges("local-cluster/777016fe-1931-4e80-95d1-d51d3b936e24", nodeStore)
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge but got %d", len(edges))
	}

	edge = edges[0]
	expectedEdge = Edge{
		EdgeType:   "generatedBy",
		SourceUID:  "local-cluster/777016fe-1931-4e80-95d1-d51d3b936e24",
		DestUID:    "local-cluster/777738fa-0591-44da-8a06-98ea7d74a7f7",
		SourceKind: "ConfigMap",
		DestKind:   "Policy",
	}

	AssertDeepEqual("edge", edge, expectedEdge, t)
}

func TestGatekeeperMutationEdges(t *testing.T) {
	// For testing common resource
	pod := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":      "my-pod",
			"uid":       "2222",
			"namespace": "test3",
			"annotations": map[string]interface{}{
				"gatekeeper.sh/mutations": "Assign//mutation-configmap:1, AssignMetadata/hello/mutation-meta-configmap:1",
			},
		},
	}}

	// For testing generic resource
	configmap := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "zk-kafka-address",
			"uid":       "1111",
			"namespace": "test2",
			"annotations": map[string]interface{}{
				"gatekeeper.sh/mutations": "Assign//mutation-configmap:1, AssignMetadata/hello/mutation-meta-configmap:1",
			},
		},
	}}

	// Case: Namespace is "default"
	assign := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mutations.gatekeeper.sh/v1",
			"kind":       "Assign",
			"metadata": map[string]interface{}{
				"name": "mutation-configmap",
				"uid":  "683aaeb0-78b9-4d44-a737-59d621bc71f0",
			},
		},
	}

	assignMeta := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mutations.gatekeeper.sh/v1",
			"kind":       "AssignMetadata",
			"metadata": map[string]interface{}{
				"name":      "mutation-meta-configmap",
				"namespace": "hello",
				"uid":       "bc09fa5f-8ef7-4cd1-833a-4c79c26b03f3",
			},
		},
	}

	nodes := []Node{
		GenericResourceBuilder(assign).BuildNode(),
		GenericResourceBuilder(assignMeta).BuildNode(),
		GenericResourceBuilder(configmap).BuildNode(),
		GenericResourceBuilder(pod).BuildNode(),
	}
	nodeStore := BuildFakeNodeStore(nodes)

	t.Log("Test Gatekeeper mutation")
	configMapEdges := CommonEdges("local-cluster/1111", nodeStore)
	if len(configMapEdges) != 2 {
		t.Fatalf("Expected 2 edge but got %d", len(configMapEdges))
	}

	podEdges := CommonEdges("local-cluster/1111", nodeStore)
	if len(configMapEdges) != 2 {
		t.Fatalf("Expected 2 edge but got %d", len(configMapEdges))
	}

	edge := configMapEdges[0]
	edge2 := podEdges[0]
	expectedEdge := Edge{
		EdgeType:   "mutatedBy",
		SourceUID:  "local-cluster/1111",
		DestUID:    "local-cluster/683aaeb0-78b9-4d44-a737-59d621bc71f0",
		SourceKind: "ConfigMap",
		DestKind:   "Assign",
	}

	AssertDeepEqual("edge", edge, expectedEdge, t)
	AssertDeepEqual("edge", edge2, expectedEdge, t)

	edge = configMapEdges[1]
	edge2 = podEdges[1]
	expectedEdge = Edge{
		EdgeType:   "mutatedBy",
		SourceUID:  "local-cluster/1111",
		DestUID:    "local-cluster/bc09fa5f-8ef7-4cd1-833a-4c79c26b03f3",
		SourceKind: "ConfigMap",
		DestKind:   "AssignMetadata",
	}

	AssertDeepEqual("edge", edge, expectedEdge, t)
	AssertDeepEqual("edge", edge2, expectedEdge, t)
}

func TestMemoryToBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int64
		shouldError bool
	}{
		{"k", "1k", 1000, false},                     // SI unit (1K = 1000 bytes)
		{"Ki", "1Ki", 1024, false},                   // Binary unit (1Ki = 1024 bytes)
		{"G", "1G", 1000000000, false},               // SI unit (1G = 1,000,000,000 bytes)
		{"Gi", "1Gi", 1073741824, false},             // Binary unit (1Gi = 1024^3 bytes)
		{"Small Value", "512Mi", 536870912, false},   // 512Mi = 512 * 1024^2
		{"Large Value", "2Ti", 2199023255552, false}, // 2Ti = 2 * 1024^4
		{"No Unit", "1024", 1024, false},             // Assumes bytes
		{"Invalid Input", "abcMB", 0, true},          // Invalid string
		{"Empty String", "", 0, true},                // Edge case: empty input
	}

	for _, tt := range tests {
		result, err := memoryToBytes(tt.input)
		assert.Equal(t, tt.shouldError, err != nil)
		assert.Equal(t, tt.expected, result)
	}
}
