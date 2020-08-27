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
	gvr               schema.GroupVersionResource
	AddFunc           func(interface{})
	DeleteFunc        func(interface{})
	UpdateFunc        func(prev interface{}, next interface{}) // We don't use prev, but matching client-go informer.
	prevResourceIndex map[string]string
	resourceIndex     map[string]string // Keeps an index of resources. key=UUID  value=resourceVersion
	retries           int64             // Counts times we have retried without establishing a successful watch.
	stopped           bool
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
			// TODO: Need exponential backoff and max wait.
			wait := time.Duration(inform.retries) * time.Second
			glog.Infof("Waiting %s before retrying listAndWatch for %s", wait, inform.gvr.String())
			time.Sleep(wait)
		}
		glog.Info("Starting informer ", inform.gvr.String())
		// glog.Info("  Existing Resources: ", inform.resourceIndex)
		listAndWatch(inform, stopper)

		if inform.stopped {
			glog.Info("Informer was stopped. ", inform.gvr.String())
			return
		}

	}

}

func listAndWatch(inform *GenericInformer, stopper chan struct{}) {
	client := config.GetDynamicClient()

	if len(inform.resourceIndex) > 0 {
		inform.prevResourceIndex = inform.resourceIndex
		inform.resourceIndex = make(map[string]string)
	}

	// 1. List and add existing resources.
	resources, listError := client.Resource(inform.gvr).List(metav1.ListOptions{})
	if listError != nil {
		glog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
		inform.retries++
		return
	}
	for i := range resources.Items {
		glog.V(5).Infof("KIND: %s UUID: %s, ResourceVersion: %s", inform.gvr.Resource, resources.Items[i].GetUID(), resources.Items[i].GetResourceVersion())
		inform.AddFunc(&resources.Items[i])
		inform.resourceIndex[string(resources.Items[i].GetUID())] = resources.Items[i].GetResourceVersion()
	}
	glog.V(3).Infof("Listed   [Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
		inform.gvr.Group, inform.gvr.Resource, len(resources.Items), resources.GetResourceVersion())

	for key := range inform.prevResourceIndex {
		if _, exist := inform.resourceIndex[key]; !exist {
			glog.Infof("!!! TODO: Need to delete resource %s with UID: %s", inform.gvr.Resource, key)
		}
	}

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
