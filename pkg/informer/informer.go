// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type GenericInformer struct {
	gvr        schema.GroupVersionResource
	AddFunc    func(interface{})
	UpdateFunc func(interface{}, interface{})
	DeleteFunc func(interface{})
	// eventHandlers cache.ResourceEventHandlerFuncs
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
		// i.eventHandlers.AddFunc(&r)
		i.AddFunc(&r)
		glog.Infof("Called AddFunc() for [ Kind: %s  Name: %s ]", r.GetKind(), r.GetName())
	}

	// TODO: Record and track the UID and current ResourceVersion.
	glog.Infof("Group: %s  Kind: %s, last resourceVersion: %s", i.gvr.Group, i.gvr.Resource, resources.GetResourceVersion())

	// 2. Start a watcher starting from resourceVersion.
	watch, watchError := client.Resource(i.gvr).Watch(metav1.ListOptions{})
	if watchError != nil {
		glog.Warningf("Error watching resources for %s.  Error: %s", i.gvr.String(), watchError)
	}
	glog.Infof("Watching Kind: %s ===> Watch: %s", i.gvr.Resource, watch)

	//  TODO: Call Add/Update/Delete functions.
	// 	TODO: Keep track of UID and current ResourceVersion.
	//	TODO: Continuously monitor the status of the watch, if it times out or connection drops, restart the watcher.

}
