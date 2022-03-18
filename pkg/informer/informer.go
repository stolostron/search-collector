// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/config"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var contextVar = context.TODO()

// GenericInformer ...
type GenericInformer struct {
	client        dynamic.Interface
	gvr           schema.GroupVersionResource
	AddFunc       func(interface{})
	DeleteFunc    func(interface{})
	UpdateFunc    func(prev interface{}, next interface{}) // We don't use prev, but matching client-go informer.
	initialized   bool
	resourceIndex map[string]string // Index of curr resources [key=UUID value=resourceVersion]
	retries       int64             // Counts times we have tried without establishing a watch.
}

type Resource struct {
	ApiGroups []string `yaml:"apiGroups"`
	Resources []string `yaml:"resources"`
}

func GetAllowDenyData(cm *v1.ConfigMap) ([]Resource, []Resource) {

	var allow []Resource
	allowerr := yaml.Unmarshal([]byte(cm.Data["AllowedResources"]), &allow)
	if allowerr != nil {
		klog.Error("Unmarshal: %v", allowerr)
	}

	var deny []Resource
	denyerr := yaml.Unmarshal([]byte(cm.Data["DeniedResources"]), &deny)
	if denyerr != nil {
		klog.Error("Unmarshal: %v", denyerr)
	}

	return allow, deny
}

// InformerForResource initialize a Generic Informer for a resource (GVR).
func InformerForResource(res schema.GroupVersionResource) (GenericInformer, error) {
	i := GenericInformer{
		gvr:           res,
		AddFunc:       (func(interface{}) { glog.Warning("AddFunc not initialized for ", res.String()) }),
		DeleteFunc:    (func(interface{}) { glog.Warning("DeleteFunc not initialized for ", res.String()) }),
		UpdateFunc:    (func(interface{}, interface{}) { glog.Warning("UpdateFunc not init for ", res.String()) }),
		initialized:   false,
		retries:       0,
		resourceIndex: make(map[string]string),
	}
	return i, nil
}

func isResourceAllowed(cm *v1.ConfigMap, group, kind string, allowedList []Resource, deniedList []Resource) bool {

	var boolVar bool

	// TODO: Use env variable for ignored resource kinds.
	// Ignore clusters and clusterstatus resources because these are handled by the aggregator.
	// Ignore oauthaccesstoken resources because those cause too much noise on OpenShift clusters.
	// Ignore projects as namespaces are overwritten to be projects on Openshift clusters - they tend to share
	// the same uid.
	list := []string{"events", "projects", "clusters", "clusterstatuses", "oauthaccesstokens"}

	//remove all apiResources with kind in list
	for _, name := range list {
		if kind == name {
			// klog.Info("list of names, resource kind:", name, kind)
			return false
		}
	}

	// remove denied resources from all apigroups if * otherwise remove from specific apigroups.
	for _, deny := range deniedList {
		for _, api := range deny.ApiGroups {
			for _, rec := range deny.Resources {
				if api == "*" && rec != "*" {
					if rec == kind {
						klog.Info("deny all group, specific resource", group, api, kind, rec)

						return false
					} else {
						if api != "*" && rec == "*" {
							if group == api {
								klog.Info("deny specific group all resources", group, api, kind, rec)
								return false
							}
						}

					}
				} else {
					// if api != "*" && deny.Resources[i] != "*" {
					if group == api && kind == rec {
						klog.Info("deny specific group specific resources", group, api, kind, rec)
						return false
					}
					// }
				}
			}
		}
	}

	// if allowedlist is empty allow all resources, otherwise if * allow all groups specific
	// resources if not * allow specific resources to specific groups:
	if len(allowedList) == 0 {
		return true
	} else {
		// klog.Info("In-coming ApiResource", kind)
		for _, allow := range allowedList {
			for _, api := range allow.ApiGroups {
				for _, rec := range allow.Resources {
					if api == "*" && rec != "*" { //all apigroups & specific resources
						if rec == kind {
							klog.Info("allow all group specific resources:", group, api, kind, rec)
							return true

						} else {
							if api != "*" && rec == "*" { // specific apigroups & all resources
								if group == api {
									klog.Info("allow specific groups all resources:", group, api, kind, rec)
									return true
								}
							}
						}

					} else {
						// if api != "*" && allow.Resources[i] != "*" { //specific apigroups and resources
						if group == api && kind == rec {
							klog.Info("allow specific group specific resources", group, api, kind, rec)
							return true
						}
						// }
					}
				}
			}
		}
	}

	return boolVar
}

// Returns a map containing all the GVRs on the cluster of resources that support WATCH (ignoring clusters and events).
func SupportedResources(discoveryClient *discovery.DiscoveryClient) (map[schema.GroupVersionResource]struct{}, error) {
	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources() //<--here we can look into this list and have preferred versions
	if err != nil && apiResources == nil {                          // only return if the list is empty
		return nil, err
	} else if err != nil {
		glog.Warning("ServerPreferredResources could not list all available resources: ", err)
	}

	// create client to get configmap
	config := config.GetKubeConfig() //can't err here?
	clientset, err := kubernetes.NewForConfig(config)

	//locate the allow-deny ConfigMap:
	var cm *v1.ConfigMap
	if cm, err = clientset.CoreV1().ConfigMaps("open-cluster-management").Get(contextVar, "allowdeny-config", metav1.GetOptions{}); err != nil {
		glog.Warning("Can't find allow/deny ConfigMap", err)
	}

	//parse config:
	allowedList, deniedList := GetAllowDenyData(cm)

	tr.NonNSResourceMap = make(map[string]struct{}) //map to store non-namespaced resources

	// Filter down to only resources which support WATCH operations
	for _, apiList := range apiResources { // This comes out in a nested list, so loop through a couple things
		// This is a copy of apiList but we only insert resources for which GET is supported.
		watchList := machineryV1.APIResourceList{}
		watchList.GroupVersion = apiList.GroupVersion
		watchResources := []machineryV1.APIResource{} // All the resources for which GET works.

		for _, apiResource := range apiList.APIResources { // Loop across inner list

			if !isResourceAllowed(cm, apiResource.Group, apiResource.Name, allowedList, deniedList) {
				continue // Skip the resource before starting the informer
			}

			// add non-namespaced resource to NonNSResourceMap
			if !apiResource.Namespaced {
				tr.NonNSResMapMutex.Lock()
				if _, ok := tr.NonNSResourceMap[apiResource.Kind]; !ok {
					tr.NonNSResourceMap[apiResource.Kind] = struct{}{}
				}
				tr.NonNSResMapMutex.Unlock()

			}
			for _, verb := range apiResource.Verbs {
				if verb == "watch" {
					watchResources = append(watchResources, apiResource)
				}
			}
		}

		watchList.APIResources = watchResources
		supportedResources = append(supportedResources, &watchList)
		// fmt.Println(supportedResources)

	}

	// Use handy converter function to convert into GroupVersionResource objects, which we need in order to make informers
	gvrList, err := discovery.GroupVersionResources(supportedResources)

	return gvrList, err
}

// Run runs the informer.
func (inform *GenericInformer) Run(stopper chan struct{}) {
	for {
		select {
		case <-stopper:
			glog.Info("Informer stopped. ", inform.gvr.String())
			return
		default:
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

			err := inform.listAndResync()
			if err == nil {
				inform.initialized = true
				inform.watch(stopper)
			}
		}
	}
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
func (inform *GenericInformer) listAndResync() error {

	// Keep track of new resources added to consolidate against the previous state.
	newResourceIndex := make(map[string]string)

	// List resources.
	opts := metav1.ListOptions{Limit: 250}
	for {
		resources, listError := inform.client.Resource(inform.gvr).List(contextVar, opts)
		if listError != nil {
			glog.Warningf("Error listing resources for %s.  Error: %s", inform.gvr.String(), listError)
			inform.retries++
			return listError
		}

		// Add all resources. <-- this is where we should change to only add resources in Allow list:
		for i := range resources.Items {
			glog.V(5).Infof("KIND: %s UUID: %s, ResourceVersion: %s",
				inform.gvr.Resource, resources.Items[i].GetUID(), resources.Items[i].GetResourceVersion())
			inform.AddFunc(&resources.Items[i])
			newResourceIndex[string(resources.Items[i].GetUID())] = resources.Items[i].GetResourceVersion()
		}
		glog.V(3).Infof("Listed\t[Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
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
			glog.V(3).Infof("Resource does not exist. Deleting resource: %s with UID: %s", inform.gvr.Resource, key)
			obj := newUnstructured(inform.gvr.Resource, key)
			inform.DeleteFunc(obj)
			delete(inform.resourceIndex, key) // Thread safe?
		}
	}
	return nil
}

// Watch resources and process events.
func (inform *GenericInformer) watch(stopper chan struct{}) {

	watch, watchError := inform.client.Resource(inform.gvr).Watch(contextVar, metav1.ListOptions{})
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
			glog.V(2).Info("Informer watch() was stopped. ", inform.gvr.String())
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
				glog.V(2).Infof("Received ERROR event. Ending listAndWatch() for %s event: %s", inform.gvr.String(), event)
				return

			default:
				glog.V(2).Infof("Received unexpected event. Ending listAndWatch() for %s", inform.gvr.String())
				return
			}
		}
	}
}

// Waits until informer has completed the initial listAndSync() of resources
// or until timeout.
func (inform *GenericInformer) WaitUntilInitialized(timeout time.Duration) {
	start := time.Now()
	for !inform.initialized {
		if time.Since(start) > timeout {
			glog.V(2).Infof("Informer [%s] timed out after %s waiting for initialization.", inform.gvr.String(), timeout)
			break
		}
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}
