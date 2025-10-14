// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	initialized   atomic.Bool
	resourceIndex map[string]string // Index of curr resources [key=UUID value=resourceVersion]
	retries       int64             // Counts times we have tried without establishing a watch.
}

// InformerForResource initialize a Generic Informer for a resource (GVR).
func InformerForResource(res schema.GroupVersionResource) (GenericInformer, error) {
	i := GenericInformer{
		gvr:           res,
		AddFunc:       (func(interface{}) { klog.Warning("AddFunc not initialized for ", res.String()) }),
		DeleteFunc:    (func(interface{}) { klog.Warning("DeleteFunc not initialized for ", res.String()) }),
		UpdateFunc:    (func(interface{}, interface{}) { klog.Warning("UpdateFunc not init for ", res.String()) }),
		retries:       0,
		resourceIndex: make(map[string]string),
	}
	return i, nil
}

// Run runs the informer.
func (inform *GenericInformer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Info("Informer stopped. ", inform.gvr.String())
			for key := range inform.resourceIndex {
				klog.V(5).Infof("Stopping informer %s and removing resource with UID: %s", inform.gvr.Resource, key)
				obj := newUnstructured(inform.gvr.Resource, key)
				inform.DeleteFunc(obj)
			}
			klog.V(5).Info("Informer stopped. ", inform.gvr.String())
			return
		default:
			if inform.retries > 0 {
				// Backoff strategy: Adds 2 seconds each retry, up to 2 mins.
				wait := time.Duration(min(inform.retries*2, 120)) * time.Second
				klog.V(3).Infof("Waiting %s before retrying listAndWatch for %s", wait, inform.gvr.String())
				time.Sleep(wait)
			}
			klog.V(3).Info("(Re)starting informer: ", inform.gvr.String())
			if inform.client == nil {
				inform.client = config.GetDynamicClient()
			}

			err := inform.listAndResync()
			if err == nil {
				inform.initialized.Store(true)
				inform.watch(ctx.Done())
			}
		}
	}
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
func (inform *GenericInformer) listAndResync() error {

	// Keep track of new resources added to consolidate against the previous state.
	newResourceIndex := make(map[string]string)

	// We need this limit to avoid a memory spike. Smaller chunks allows us to release memory faster, however
	// it generates more requests to the kube api server.
	opts := metav1.ListOptions{Limit: 250}
	for {
		resources, listError := inform.client.Resource(inform.gvr).List(context.TODO(), opts)
		if listError != nil {
			klog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
			inform.retries++
			return listError
		}

		// Add all resources.
		for i := range resources.Items {
			klog.V(5).Infof("KIND: %s UUID: %s, ResourceVersion: %s",
				inform.gvr.Resource, resources.Items[i].GetUID(), resources.Items[i].GetResourceVersion())
			inform.AddFunc(&resources.Items[i])
			newResourceIndex[string(resources.Items[i].GetUID())] = resources.Items[i].GetResourceVersion()
		}
		klog.V(3).Infof("Listed\t[Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
			inform.gvr.Group, inform.gvr.Resource, len(resources.Items), resources.GetResourceVersion())

		// Check if there's more items and set the "continue" option for the next request.
		// If there isn't any more items we break from the loop.
		metadata := resources.UnstructuredContent()["metadata"].(map[string]interface{})
		if metadata["remainingItemCount"] != nil && metadata["remainingItemCount"] != 0 {
			opts.Continue = metadata["continue"].(string)
		} else {
			break
		}
	}

	// Delete resources from previous state that no longer exist in the new state.
	for key := range inform.resourceIndex {
		if _, exist := newResourceIndex[key]; !exist {
			klog.V(3).Infof("Resource does not exist. Deleting resource: %s with UID: %s", inform.gvr.Resource, key)
			obj := newUnstructured(inform.gvr.Resource, key)
			inform.DeleteFunc(obj)
			delete(inform.resourceIndex, key)
		}
	}
	return nil
}

// Watch resources and process events.
func (inform *GenericInformer) watch(stopper <-chan struct{}) {
	watch, watchError := inform.client.Resource(inform.gvr).Watch(context.TODO(), metav1.ListOptions{})
	if watchError != nil {
		klog.Warningf("Error watching resources for %s.  Error: %s", inform.gvr.String(), watchError)
		inform.retries++
		return
	}
	defer watch.Stop()

	klog.V(3).Infof("Watching\t[Group: %s \tKind: %s]", inform.gvr.Group, inform.gvr.Resource)

	watchEvents := watch.ResultChan()
	inform.retries = 0 // Reset retries because we have a successful list and a watch.

	for {
		select {
		case <-stopper:
			klog.V(2).Info("Informer watch() was stopped. ", inform.gvr.String())
			return

		case event := <-watchEvents: // Read events from the watch channel.
			//  Process ADDED, MODIFIED, DELETED, and ERROR events.
			switch event.Type {
			case "ADDED":
				klog.V(5).Infof("Received ADDED event. Kind: %s ", inform.gvr.Resource)
				obj, ok := event.Object.(*unstructured.Unstructured)
				if !ok {
					klog.Warningf("Error converting %s event.Object to unstructured.Unstructured on ADDED event.",
						inform.gvr.Resource)
					continue
				}
				inform.AddFunc(obj)
				inform.resourceIndex[string(obj.GetUID())] = obj.GetResourceVersion()

			case "MODIFIED":
				klog.V(5).Infof("Received MODIFY event. Kind: %s ", inform.gvr.Resource)
				obj, ok := event.Object.(*unstructured.Unstructured)
				if !ok {
					klog.Warningf("Error converting %s event.Object to unstructured.Unstructured on MODIFIED event",
						inform.gvr.Resource)
					continue
				}

				inform.UpdateFunc(nil, obj)
				inform.resourceIndex[string(obj.GetUID())] = obj.GetResourceVersion()

			case "DELETED":
				klog.V(5).Infof("Received DELETED event. Kind: %s ", inform.gvr.Resource)
				obj, ok := event.Object.(*unstructured.Unstructured)
				if !ok {
					klog.Warningf("Error converting %s event.Object to unstructured.Unstructured on DELETED event",
						inform.gvr.Resource)
					continue
				}

				inform.DeleteFunc(obj)
				delete(inform.resourceIndex, string(obj.GetUID()))

			case "ERROR":
				klog.V(2).Infof("Received ERROR event. Ending listAndWatch() for %s event: %s", inform.gvr.String(), event)
				return

			default:
				klog.V(2).Infof("Received unexpected event. Ending listAndWatch() for %s", inform.gvr.String())
				return
			}
		}
	}
}

// Waits until informer has completed the initial listAndSync() of resources
// or until timeout.
func (inform *GenericInformer) WaitUntilInitialized(timeout time.Duration) {
	start := time.Now()
	for !inform.initialized.Load() {
		if time.Since(start) > timeout {
			klog.V(2).Infof("Informer [%s] timed out after %s waiting for initialization.", inform.gvr.String(), timeout)
			break
		}
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}
