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
	"math"
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

func TestGatekeeperMutationEdges(t *testing.T) {
	config.InitConfig()

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

func TestGetConditionsNestedSliceFail(t *testing.T) {
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
			"status": map[string]interface{}{
				"conditions": "not-a-slice", // this should cause NestedSlice to return an error
			},
		},
	}

	result, err := getConditions(resource)
	assert.Error(t, err, "Expected error when status.conditions is not a slice")
	assert.Nil(t, result, "Expected nil result when error occurs")
}

func TestGetConditionsMarshalConditionsFail(t *testing.T) {
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Ready",
						"status":             "True",
						"lastTransitionTime": "2024-01-01T00:00:00Z",
						"invalidFloat":       math.Inf(1), // this should cause json.Marshal to fail
					},
				},
			},
		},
	}

	result, err := getConditions(resource)
	assert.Error(t, err, "Expected error when marshaling fails due to Infinity value")
	assert.Nil(t, result, "Expected nil result when marshal error occurs")
}

func TestGetConditionsUnmarshalFail(t *testing.T) {
	resource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
						"lastTransitionTime": map[string]interface{}{
							"invalid": "structure", // this should fail to unmarshal into time
						},
					},
				},
			},
		},
	}

	result, err := getConditions(resource)
	assert.Error(t, err, "Expected error when unmarshaling fails")
	assert.Nil(t, result, "Expected nil result when unmarshal error occurs")
}

func Test_genericResourceFromConfigMapNoLabelsToMatchMatchLabel(t *testing.T) {
	config.Cfg.DeployedInHub = false // temporarily set to false else _hubClusterResource gets appended during full test suite
	defer func() {
		config.Cfg.DeployedInHub = true
	}()
	var r unstructured.Unstructured
	UnmarshalFile("configmap-two.json", &r, t)
	// when matchLabel checks against nil labels we skip the configured additional ExtractProperties
	r.SetLabels(nil)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "app-config", t)
	AssertEqual("kind", node.Properties["kind"], "ConfigMap", t)
	AssertEqual("created", node.Properties["created"], "2026-01-05T14:27:31Z", t)
	AssertEqual("apiversion", node.Properties["apiversion"], "v1", t)
	AssertEqual("namespace", node.Properties["namespace"], "default", t)

	// Verify that there's no more indexed properties than the common ones
	assert.Equal(t, 5, len(node.Properties))
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		success  bool
	}{
		// String types
		{"String", "hello", "hello", true},
		{"Empty String", "", "", true},

		// Boolean types
		{"Bool True", true, "true", true},
		{"Bool False", false, "false", true},

		// Integer types
		{"Int", 42, "42", true},
		{"Int Negative", -100, "-100", true},
		{"Int Zero", 0, "0", true},
		{"Int64", int64(9223372036854775807), "9223372036854775807", true},
		{"Int64 Negative", int64(-9223372036854775808), "-9223372036854775808", true},

		// Float types
		{"Float64 Integer Value", 100.0, "100", true},         // Should format as integer
		{"Float64 With Decimals", 3.14159, "3.14159", true},   // Should format as float
		{"Float64 Negative", -2.5, "-2.5", true},              // Negative float
		{"Float64 Zero", 0.0, "0", true},                      // Zero as integer
		{"Float64 Large Integer", 1000000.0, "1000000", true}, // Large integer value
		{"Float64 Small Decimal", 0.001, "0.001", true},       // Small decimal

		// Unsupported types
		{"Nil", nil, "", false},
		{"Slice", []string{"a", "b"}, "", false},
		{"Map", map[string]string{"key": "value"}, "", false},
		{"Struct", struct{ Name string }{Name: "test"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := convertToString(tt.input)
			assert.Equal(t, tt.success, ok, "Success flag mismatch for %s", tt.name)
			if tt.success {
				assert.Equal(t, tt.expected, result, "Result mismatch for %s", tt.name)
			}
		})
	}
}
