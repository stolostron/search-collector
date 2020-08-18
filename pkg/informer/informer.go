// Copyright (c) 2020 Red Hat, Inc.

package informer

import (
	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
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
	}
	return i, nil
}

func (i GenericInformer) AddEventHandler(h cache.ResourceEventHandlerFuncs) {
	i.AddFunc = h.AddFunc
	i.UpdateFunc = h.UpdateFunc
	i.DeleteFunc = h.DeleteFunc

}

func (i GenericInformer) Run(stopper chan struct{}) {
	glog.Info("Starting informer ", i.gvr.String())

	// 1. List all resources for a given GVR (GroupVersionResource)
	client := config.GetDynamicClient()
	resources, listError := client.Resource(i.gvr).List(metav1.ListOptions{})

	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", i.gvr.String(), listError)
	}
	// TODO: For each resource invoke AddFunc()
	// TODO: Record and track the UID and current ResourceVersion.
	glog.Info("Resources:  ", resources)

	// 2. Start a watcher starting from resourceVersion.
	watch, watchError := client.Resource(i.gvr).Watch(metav1.ListOptions{})

	glog.Info("Watch:", watch, watchError)
	//   	Call Add/Update/Delete functions.
	// 		Keep track of UID and current ResourceVersion.
	//		Continuously monitor the status of the watch, if it times out or connection drops, restart the watcher.

}
