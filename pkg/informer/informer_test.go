// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func generateSimpleEvent(informer GenericInformer, t *testing.T) {
	// Add resource. Generates ADDED event.
	gvr := schema.GroupVersionResource{Group: "open-cluster-management.io", Version: "v1", Resource: "thekinds"}
	newResource := newTestUnstructured("open-cluster-management.io/v1", "TheKind", "ns-foo", "name-new", "id-999")
	_, err1 := informer.client.Resource(gvr).Namespace("ns-foo").Create(newResource, v1.CreateOptions{})

	// Update resource. Generates MODIFIED event.
	_, err2 := informer.client.Resource(gvr).Namespace("ns-foo").Update(newResource, v1.UpdateOptions{})

	// Delete resource. Generated DELETED event.
	err3 := informer.client.Resource(gvr).Namespace("ns-foo").Delete("name-bar2", &v1.DeleteOptions{})

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("Error generating mocked events.")
	}
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

func initInformer() (informer GenericInformer, _ *int, _ *int, _ *int) {
	// Create informer instance to test.
	gvr := schema.GroupVersionResource{Group: "open-cluster-management.io", Version: "v1", Resource: "thekinds"}
	informer, _ = InformerForResource(gvr)

	// Add the fake client to be used by informer.
	informer.client = fakeDynamicClient()

	// Add mock functions
	var addFuncCount, updateFuncCount, deleteFuncCount int
	informer.AddFunc = func(interface{}) { addFuncCount++ }
	informer.DeleteFunc = func(interface{}) { deleteFuncCount++ }
	informer.UpdateFunc = func(interface{}, interface{}) { updateFuncCount++ }

	return informer, &addFuncCount, &deleteFuncCount, &updateFuncCount
}

// Verify that AddFunc is called for each mocked resource.
func Test_listAndResync(t *testing.T) {

	// Create informer instance to test.
	informer, addFuncCount, _, _ := initInformer()

	// Execute function
	informer.listAndResync()

	// Verify that informer.AddFunc is called for each of the mocked resources (5 times).
	if *addFuncCount != 5 {
		t.Errorf("Expected informer.AddFunc to be called 5 times, but got %d.", *addFuncCount)
	}
}

// Verify that DeleteFunc is called for indexed resources that no longer exist.
func Test_listAndResync_syncWithPrevState(t *testing.T) {
	// Create informer instance to test.
	informer, _, deleteFuncCount, _ := initInformer()

	// Add existing state to the informer
	informer.resourceIndex["fake-uid"] = "fake-resource-version" // This resource should get deleted.
	informer.resourceIndex["id-001"] = "some-resource-version"   // This resource won't get deleted.

	// Execute function
	informer.listAndResync()

	// Verify that informer.DeleteFunc is called once for resource with "fake-uid"
	if *deleteFuncCount != 1 {
		t.Errorf("Expected informer.DeleteFunc to be called 1 time, but got %d.", *deleteFuncCount)
	}
}

// Verify the informer's Run function.
func Test_Run(t *testing.T) {
	// Create informer instance to test.
	informer, addFuncCount, deleteFuncCount, updateFuncCount := initInformer()

	// Start informer routine
	stopper := make(chan struct{})
	go informer.Run(stopper)
	time.Sleep(10 * time.Millisecond)

	generateSimpleEvent(informer, t)
	time.Sleep(10 * time.Millisecond)

	stopper <- struct{}{}

	// Verify that informer.AddFunc is called for each of the mocked resources (6 times).
	if *addFuncCount != 6 {
		t.Errorf("Expected informer.AddFunc to be called 6 times, but got %d.", *addFuncCount)
	}
	// Verify informer.UpdateFunc is called once.
	if *updateFuncCount != 1 {
		t.Errorf("Expected informer.UpdateFunc to be called 1 times, but got %d.", *updateFuncCount)
	}
	// Verify informer.DeleteFunc is called once.
	if *deleteFuncCount != 1 {
		t.Errorf("Expected informer.DeleteFunc to be called 1 times, but got %d.", *deleteFuncCount)
	}
}

// Verify that backoff logic waits after retry.
func Test_Run_retryBackoff(t *testing.T) {
	// Create informer instance to test.
	informer, _, _, _ := initInformer()

	informer.retries = 1
	startTime := time.Now()
	retryTime := time.Now() // Initializing to now ensures that the test fail if AddFunc is not called in the expected time.
	informer.AddFunc = func(interface{}) { retryTime = time.Now() }

	// Execute function
	go informer.Run(make(chan struct{}))
	time.Sleep(2010 * time.Millisecond)

	// Verify backoff logic waits 2 seconds before retrying.
	if startTime.Add(2 * time.Second).After(retryTime) {
		t.Errorf("Backoff logic failed to wait for 2 seconds.")
	}
}

// Verify that the informer is able to watch resources and process the events.
func Test_watch(t *testing.T) {
	// Create informer instance to test.
	informer, _, _, _ := initInformer()

	// Create a stopper for the watch function
	stopper := make(chan struct{})

	go informer.watch(stopper)
	time.Sleep(10 * time.Millisecond)

	generateSimpleEvent(informer, t)
	time.Sleep(10 * time.Millisecond)

	// Simulate that the informer has been stopped successfully.
	stopper <- struct{}{}
	time.Sleep(10 * time.Millisecond)

	if !informer.stopped {
		t.Errorf("Expected informer.stopped to be true, but got %t", informer.stopped)
	}
}
