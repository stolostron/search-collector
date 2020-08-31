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

type GenericInformer struct {
	gvr           schema.GroupVersionResource
	AddFunc       func(interface{})
	DeleteFunc    func(interface{})
	UpdateFunc    func(prev interface{}, next interface{}) // We don't use prev, but matching client-go informer.
	resourceIndex map[string]string                        // Keeps an index of resources. key=UUID  value=resourceVersion
	retries       int64                                    // Counts times we have retried without establishing a successful watch.
	stopped       bool                                     // Tracks when the informer is stopped, to exit cleanly.
}

func InformerForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	i := GenericInformer{
		gvr: resource,
		AddFunc: (func(interface{}) {
			glog.Info("Add function not initialized.")
		}),
		UpdateFunc: (func(interface{}, interface{}) {
			glog.Info("Update function not initialized.")
		}),
		DeleteFunc: (func(interface{}) {
			glog.Info("Delete function not initialized.")
		}),
		retries:       0,
		resourceIndex: make(map[string]string),
	}
	return i, nil
}

func (inform *GenericInformer) Run(stopper chan struct{}) {
	for {
		if inform.retries > 0 {
			// Backoff strategy: Adds 2 seconds each retry, up to 2 mins.
			wait := time.Duration(min(inform.retries*2, 120)) * time.Second
			glog.V(3).Infof("Waiting %s before retrying listAndWatch for %s", wait, inform.gvr.String())
			time.Sleep(wait)
		}
		glog.V(2).Info("(Re)starting informer: ", inform.gvr.String())
		listAndWatch(inform, stopper)

		if inform.stopped {
			break
		}
	}
	glog.V(3).Info("Informer was stopped. ", inform.gvr.String())
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

// List resources and start a watch. When restarting, it syncs resources with the previous state.
func listAndWatch(inform *GenericInformer, stopper chan struct{}) {
	client := config.GetDynamicClient()

	listAndResync(inform, client)

	watch(inform, client, stopper)
}

// List current resources and fires ADDED events. Then sync the current state with the previous
// state and delete any resources that are still in our cache, but no longer exist in the cluster.
func listAndResync(inform *GenericInformer, client dynamic.Interface) {
	// Save the previous state.
	var prevResourceIndex map[string]string
	if len(inform.resourceIndex) > 0 {
		prevResourceIndex = inform.resourceIndex
		inform.resourceIndex = make(map[string]string)
	}

	// List resources.
	resources, listError := client.Resource(inform.gvr).List(metav1.ListOptions{})
	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
		inform.retries++
		return
	}

	// Add all resources.
	for i := range resources.Items {
		glog.V(5).Infof("KIND: %s UUID: %s, ResourceVersion: %s", inform.gvr.Resource, resources.Items[i].GetUID(), resources.Items[i].GetResourceVersion())
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
			break
		}
	}
}

// Watch resources and process events.
func watch(inform *GenericInformer, client dynamic.Interface, stopper chan struct{}) {

	watch, watchError := client.Resource(inform.gvr).Watch(metav1.ListOptions{})
	if watchError != nil {
		glog.Warningf("Error watching resources for %s.  Error: %s", inform.gvr.String(), watchError)
		inform.retries++
		return
	}
	glog.V(3).Infof("Watching\t[Group: %s \tKind: %s]  ===>  Watch: %s", inform.gvr.Group, inform.gvr.Resource, watch)

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
				glog.Warningf("Received ERROR event. Ending listAndWatch() for %s ", inform.gvr.String())
				glog.Warning("  Event: ", event)
				watch.Stop()
				return

			default:
				glog.Warningf("Received unexpected event. Ending listAndWatch() for %s ", inform.gvr.String())
				watch.Stop()
				return
			}
		}
	}
}
