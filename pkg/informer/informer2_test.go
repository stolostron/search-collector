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
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-foo2", "id-002"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-bar", "id-003"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-bar2", "id-004"),
		newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-bar3", "id-005"),
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

// Verify that AddFunc is called for each mocked resource.
func Test_listAndResync(t *testing.T) {

	// Create informer instance to test.
	gvr := schema.GroupVersionResource{Group: "open-cluster-management.io", Version: "v1", Resource: "thekinds"}
	informer, _ := InformerForResource(gvr)

	// Add the fake client to be used by informer.
	informer.client = fakeDynamicClient()

	// Mock the AddFunc to count how many times it gets called.
	var addFunc_count = 0
	informer.AddFunc = func(interface{}) { addFunc_count++ }

	// Execute function
	informer.listAndResync()

	// Verify that informer.AddFunc is called for each of the mocked resources (5 times).
	if addFunc_count != 5 {
		t.Errorf("Expected informer.AddFunc to be called 5 times, but got %d.", addFunc_count)
	}
}

// Verify that DeleteFunc is called for indexed resources that no longer exist.
func Test_listAndResync_syncWithPrevState(t *testing.T) {

	// Create informer instance to test.
	gvr := schema.GroupVersionResource{Group: "open-cluster-management.io", Version: "v1", Resource: "thekinds"}
	informer, _ := InformerForResource(gvr)

	// Add the fake client to be used by informer.
	informer.client = fakeDynamicClient()

	// Add existing state to the informer
	informer.resourceIndex["fake-uid"] = "fake-resource-version" // This resource should get deleted.
	informer.resourceIndex["id-001"] = "some-resource-version"   // This resource won't get deleted.

	// Mock the DeleteFunc to count how many times it gets called.
	var deleteFunc_count = 0
	informer.DeleteFunc = func(interface{}) { deleteFunc_count++ }

	// Execute function
	informer.listAndResync()

	// Verify that informer.DeleteFunc is called once for resource with "fake-uid"
	if deleteFunc_count != 1 {
		t.Errorf("Expected informer.DeleteFunc to be called 1 time, but got %d.", deleteFunc_count)
	}
}
