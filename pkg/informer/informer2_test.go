// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func fakeDynamicClient() *fake.FakeDynamicClient {

	scheme := runtime.NewScheme()
	return fake.NewSimpleDynamicClient(scheme,
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-foo", "id-001"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name2-foo", "id-002"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-bar", "id-003"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-baz", "id-004"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name2-baz", "id-005"),
	)
}

func newTestUnstructured(apiVersion, kind, namespace, name, uid string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
				"uid":       uid,
			},
		},
	}
}

func Test_listAndResync(t *testing.T) {
	client := fakeDynamicClient()

	// Create informer instance to test.
	gvr := schema.GroupVersionResource{Group: "open-cluster-management.io", Version: "v1", Resource: "thekinds"}
	informer, _ := InformerForResource(gvr)

	// Mock the AddFunc to count how many times it gets called.
	var addFunc_count = 0
	informer.AddFunc = func(interface{}) { addFunc_count++ }

	// Execute function
	listAndResync(&informer, client)

	// Verify that informer.AddCunf is called for each of the mocked resources (5 times).
	if addFunc_count != 5 {
		t.Errorf("Expected informer.AddFunc to be called 5 times, but got %d.", addFunc_count)
	}
}
