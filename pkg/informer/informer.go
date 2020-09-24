// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
	"time"

	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GenericInformer ...
type GenericInformer struct {
	client        dynamic.Interface
	gvr           schema.GroupVersionResource
	AddFunc       func(interface{})
	DeleteFunc    func(interface{})
	UpdateFunc    func(prev interface{}, next interface{}) // We don't use prev, but matching client-go informer.
	resourceIndex map[string]string                        // Index of curr resources [key=UUID value=resourceVersion]
	retries       int64                                    // Counts times we have tried without establishing a watch.
	stopped       bool                                     // Tracks when the informer is stopped, used to exit cleanly
	syncCompleted bool
}

// InformerForResource initialize a Generic Informer for a resource (GVR).
func InformerForResource(res schema.GroupVersionResource) (GenericInformer, error) {
	i := GenericInformer{
		gvr:           res,
		AddFunc:       (func(interface{}) { glog.Warning("AddFunc not initialized for ", res.String()) }),
		DeleteFunc:    (func(interface{}) { glog.Warning("DeleteFunc not initialized for ", res.String()) }),
		UpdateFunc:    (func(interface{}, interface{}) { glog.Warning("UpdateFunc not init for ", res.String()) }),
		retries:       0,
		resourceIndex: make(map[string]string),
		syncCompleted: false,
	}
	return i, nil
}

// Run runs the informer.
func (inform *GenericInformer) Run(stopper chan struct{}) {
	for !inform.stopped {
		if inform.retries > 0 {
			// Backoff strategy: Adds 2 seconds each retry, up to 2 mins.
			wait := time.Duration(min(inform.retries*2, 120)) * time.Second
			glog.V(3).Infof("Waiting %s before retrying listAndWatch for %s", wait, inform.gvr.String())
			time.Sleep(wait)
		}
		glog.V(3).Info("(Re)starting informer: ", inform.gvr.String())
		if inform.client == nil {
			inform.client = config.GetDynamicClient()
		}

		inform.listAndResync()
		time.Sleep(30 * time.Second) // Try to delay until memory is released.
		inform.watch(stopper)

	}
	glog.V(2).Info("Informer was stopped. ", inform.gvr.String())
}

// Helper function that returns the smaller of two integers.
func min(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}

// Helper function that creates a new unstructured resource with given Kind and UID.
func newUnstructured(kind, uid string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": kind,
			"metadata": map[string]interface{}{
				"uid": uid,
			},
		},
	}
}

// List current resources and fires ADDED events. Then sync the current state with the previous
// state and delete any resources that are still in our cache, but no longer exist in the cluster.
func (inform *GenericInformer) listAndResync() {
	inform.syncCompleted = false

	// List resources.
	resources, listError := inform.client.Resource(inform.gvr).List(metav1.ListOptions{})
	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
		inform.retries++
		return
	}

	// Save the previous state.
	// IMPORTANT: Keep this after we have successfully listed the resources, otherwise we'll lose the previous state.
	var prevResourceIndex map[string]string
	if len(inform.resourceIndex) > 0 {
		prevResourceIndex = inform.resourceIndex
		inform.resourceIndex = make(map[string]string)
	}

	// Add all resources.
	for i := range resources.Items {
		glog.V(5).Infof("KIND: %s UUID: %s, ResourceVersion: %s",
			inform.gvr.Resource, resources.Items[i].GetUID(), resources.Items[i].GetResourceVersion())
		inform.AddFunc(&resources.Items[i])
		inform.resourceIndex[string(resources.Items[i].GetUID())] = resources.Items[i].GetResourceVersion()
	}
	glog.V(3).Infof("Listed\t[Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
		inform.gvr.Group, inform.gvr.Resource, len(resources.Items), resources.GetResourceVersion())

	// Delete resources from previous state that no longer exist in the current state.
	for key := range prevResourceIndex {
		if _, exist := inform.resourceIndex[key]; !exist {
			glog.V(3).Infof("Resource does not exist. Deleting resource: %s with UID: %s", inform.gvr.Resource, key)
			obj := newUnstructured(inform.gvr.Resource, key)
			inform.DeleteFunc(obj)
		}
	}

	inform.syncCompleted = true
}

// Watch resources and process events.
func (inform *GenericInformer) watch(stopper chan struct{}) {

	watch, watchError := inform.client.Resource(inform.gvr).Watch(metav1.ListOptions{})
	if watchError != nil {
		glog.Warningf("Error watching resources for %s.  Error: %s", inform.gvr.String(), watchError)
		inform.retries++
		return
	}
	defer watch.Stop()

	glog.V(3).Infof("Watching\t[Group: %s \tKind: %s]", inform.gvr.Group, inform.gvr.Resource)

	watchEvents := watch.ResultChan()
	inform.retries = 0 // Reset retries because we have a successful list and a watch.

	for {
		select {
		case <-stopper:
			inform.stopped = true
			return

		case event := <-watchEvents: // Read events from the watch channel.
			//  Process ADDED, MODIFIED, DELETED, and ERROR events.
			switch event.Type {
			case "ADDED":
				glog.V(5).Infof("Received ADDED event. Kind: %s ", inform.gvr.Resource)
				o, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
				if error != nil {
					glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on ADDED event. %s",
						inform.gvr.Resource, error)
				}
				obj := &unstructured.Unstructured{Object: o}
				inform.AddFunc(obj)
				inform.resourceIndex[string(obj.GetUID())] = obj.GetResourceVersion()

			case "MODIFIED":
				glog.V(5).Infof("Received MODIFY event. Kind: %s ", inform.gvr.Resource)
				o, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
				if error != nil {
					glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on MODIFIED event. %s",
						inform.gvr.Resource, error)
				}
				obj := &unstructured.Unstructured{Object: o}

				inform.UpdateFunc(nil, obj)
				inform.resourceIndex[string(obj.GetUID())] = obj.GetResourceVersion()

			case "DELETED":
				glog.V(5).Infof("Received DELETED event. Kind: %s ", inform.gvr.Resource)
				o, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
				if error != nil {
					glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on DELETED event. %s",
						inform.gvr.Resource, error)
				}
				obj := &unstructured.Unstructured{Object: o}

				inform.DeleteFunc(obj)
				delete(inform.resourceIndex, string(obj.GetUID()))

			case "ERROR":
				glog.V(2).Infof("Received ERROR event. Ending listAndWatch() for %s", inform.gvr.String())
				return

			default:
				glog.V(2).Infof("Received unexpected event. Ending listAndWatch() for %s", inform.gvr.String())
				return
			}
		}
	}
}

func (inform *GenericInformer) WaitForResync() {
	for !inform.syncCompleted {
		time.Sleep(time.Duration(10 * time.Millisecond))
	}
}
