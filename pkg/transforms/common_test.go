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
