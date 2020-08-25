// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
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
	UpdateFunc func(interface{}, interface{})
	DeleteFunc func(interface{})
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
	}
	return i, nil
}

func (i GenericInformer) Run(stopper chan struct{}) {
	glog.Info("Starting informer ", i.gvr.String())

	// 1. List all resources for a given GVR (GroupVersionResource)
	client := config.GetDynamicClient()
	resources, listError := client.Resource(i.gvr).List(metav1.ListOptions{})

	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", i.gvr.String(), listError)
	}
	// For each resource invoke AddFunc()
	for _, r := range resources.Items {
		i.AddFunc(&r)
		// glog.Infof("Called AddFunc() for [ Kind: %s  Name: %s ]", r.GetKind(), r.GetName())
	}

	glog.Infof("Listed   [Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s", i.gvr.Group, i.gvr.Resource, len(resources.Items), resources.GetResourceVersion())

	// 2. Start a watcher starting from resourceVersion.
	watch, watchError := client.Resource(i.gvr).Watch(metav1.ListOptions{})
	if watchError != nil {
		glog.Warningf("Error watching resources for %s.  Error: %s", i.gvr.String(), watchError)
	}
	glog.Infof("Watching [Group: %s \tKind: %s]  ===>  Watch: %s", i.gvr.Group, i.gvr.Resource, watch)

	watchEvents := watch.ResultChan()

	for {
		event := <-watchEvents // Read from the input channel

		//  Process Add/Update/Delete events.
		if event.Type == "ADDED" {
			glog.Infof("Received ADDED event. Kind: %s ", i.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on ADDED event. %s", i.gvr.Resource, error)
			}
			i.AddFunc(&unstructured.Unstructured{Object: obj})

		} else if event.Type == "MODIFIED" {
			// glog.Infof("Received MODIFY event. Kind: %s ", i.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on MODIFIED event. %s", i.gvr.Resource, error)
			}
			un := &unstructured.Unstructured{Object: obj}

			i.UpdateFunc(nil, un)
		} else if event.Type == "DELETED" {
			glog.Infof("Received DELETED event. Kind: %s ", i.gvr.Resource)
			obj, error := runtime.UnstructuredConverter.ToUnstructured(runtime.DefaultUnstructuredConverter, &event.Object)
			if error != nil {
				glog.Warningf("Error converting %s event.Object to unstructured.Unstructured on DELETED event. %s", i.gvr.Resource, error)
			}
			i.DeleteFunc(&unstructured.Unstructured{Object: obj})
		} else {
			glog.Error("ERROR: Received unexpected event. Should restart the watcher.", i.gvr.Group, i.gvr.Resource, event)
		}
	}

	// 	TODO: Keep track of UID and current ResourceVersion.
	//	TODO: Continuously monitor the status of the watch, if it times out or connection drops, restart the watcher.
}
