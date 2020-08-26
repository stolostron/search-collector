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
)

type GenericInformer struct {
	gvr        schema.GroupVersionResource
	AddFunc    func(interface{})
	DeleteFunc func(interface{})
	UpdateFunc func(prev interface{}, next interface{}) // We don't use prev, but matching client-go informer.
	retries    int64                                    // Counts times we have retried without establishing a successful watch.
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
		retries: 0,
	}
	return i, nil
}

func (inform GenericInformer) Run(stopper chan struct{}) {

	for {
		if inform.retries > 0 {
			// TODO: Need exponential backoff and max wait.
			wait := time.Duration(inform.retries) * time.Second
			glog.Infof("Waiting %s before retrying listAndWatch for %s", wait, inform.gvr.String())
			time.Sleep(wait)
		}
		glog.Info("Starting informer ", inform.gvr.String())
		listAndWatch(inform)
	}

	// TODO: Implement stopper.
	// stop := <-stopper
	// if stop != nil {
	// glog.Info("!!! Informer stopped???", stop)
	// }

	// 	TODO:
	//    - [Maybe don't need this] Keep track of UID and current ResourceVersion.
	//	  - Continuously monitor the status of the watch, if it times out or connection drops, restart the watcher.
}

func listAndWatch(inform GenericInformer) {
	client := config.GetDynamicClient()

	// 1. List and add existing resources.
	resources, listError := client.Resource(inform.gvr).List(metav1.ListOptions{})
	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
		inform.retries++
		return
	}
	for i := range resources.Items {
		inform.AddFunc(&resources.Items[i])
	}
	glog.V(3).Infof("Listed   [Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
		inform.gvr.Group, inform.gvr.Resource, len(resources.Items), resources.GetResourceVersion())

	// 2. Start a watcher starting from resourceVersion.
	watch, watchError := client.Resource(inform.gvr).Watch(metav1.ListOptions{})
	if watchError != nil {
		glog.Warningf("Error watching resources for %s.  Error: %s", inform.gvr.String(), watchError)
		inform.retries++
		return
	}
	glog.V(3).Infof("Watching [Group: %s \tKind: %s]  ===>  Watch: %s", inform.gvr.Group, inform.gvr.Resource, watch)

	watchEvents := watch.ResultChan()
	inform.retries = 0 // Reset retries because we have a successful list and a watch.
	for {

		// TODO: Implement stopper.
		// stop := <-stopper
		// if stop != nil {
		// glog.Info("!!! Informer stopped???", stop)
		// }

		event := <-watchEvents // Read events from the watch channel.

		//  Process ADDED, MODIFIED, DELETED, and ERROR events.
		switch event.Type {
		case "ADDED":
			glog.V(5).Infof("Received ADDED event. Kind: %s ", inform.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on ADDED event. %s",
					inform.gvr.Resource, error)
			}
			inform.AddFunc(&unstructured.Unstructured{Object: obj})

		case "MODIFIED":
			glog.V(5).Infof("Received MODIFY event. Kind: %s ", inform.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on MODIFIED event. %s",
					inform.gvr.Resource, error)
			}
			inform.UpdateFunc(nil, &unstructured.Unstructured{Object: obj})

		case "DELETED":
			glog.V(5).Infof("Received DELETED event. Kind: %s ", inform.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on DELETED event. %s",
					inform.gvr.Resource, error)
			}
			inform.DeleteFunc(&unstructured.Unstructured{Object: obj})

		case "ERROR":
			glog.Warningf("Received ERROR event. Ending listAndWatch() for %s ", inform.gvr.Resource)
			glog.Warning("  >>> Event: ", event)
			watch.Stop()
			return
		default:
			glog.Warningf("Received UNEXPECTED event. Ending listAndWatch() for %s ", inform.gvr.Resource)
			glog.Warning("  >>> Event:  ", event)
			watch.Stop()
			return
		}
	}
}
