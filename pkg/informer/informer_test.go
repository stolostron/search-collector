/*
Copyright (c) 2020 Red Hat, Inc.
*/
package informer

import (
	"encoding/json"
	"io/ioutil"
	"log"
	str "strings"
	"sync"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	te "k8s.io/client-go/testing"
)

var dynamicClient = fake.FakeDynamicClient{}
var gvrList = []schema.GroupVersionResource{}
var resources []*unstructured.Unstructured

// The API Resource List is retreived through the discovery client
// so since we're skipping the discovery client and just using the GVR List, we can bypass using the API List
// var apiResourceList = []v1.APIResource{}

func initAPIResources(t *testing.T) {
	dir := "../../test-data"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	var filePath string
	//Convert to events
	for _, file := range files {
		filePath = dir + "/" + file.Name()

		file, _ := ioutil.ReadFile(filePath)
		var data *unstructured.Unstructured

		_ = json.Unmarshal([]byte(file), &data)
		kind := data.GetKind()

		// Some resources kinds aren't listed within the test-data
		if kind == "" {
			continue
		}

		t.Logf("Located file for %s resource", kind)

		// newResource := v1.APIResource{}
		// newResource.Name = data.GetName()
		// newResource.Kind = kind

		apiVersion := str.Split(data.GetAPIVersion(), "/")

		// Set GVR resource to current data resource.
		var gvr schema.GroupVersionResource
		gvr.Resource = str.Join([]string{str.ToLower(kind), "s"}, "")

		// Set the GVR Group and Version
		if len(apiVersion) == 2 {
			// newResource.Group = apiVersion[0]
			// newResource.Version = apiVersion[1]

			gvr.Group = apiVersion[0]
			gvr.Version = apiVersion[1]
		} else {
			// newResource.Version = apiVersion[0]
			gvr.Version = apiVersion[0]
		}

		gvrList = append(gvrList, gvr)

		// Not all resources will be namespaced, so we need to check for that value.
		// if data.GetNamespace() != "" {
		// 	newResource.Namespaced = true
		// }

		// Set the verbs for resources.
		// newResource.Verbs = []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}

		// t.Logf("[newResource]\t ===> %+v\n\n", newResource)
		// apiResourceList = append(apiResourceList, newResource)

		// We need to create resources for the dynamic client.
		_, err := dynamicClient.Resource(gvr).Create(data, v1.CreateOptions{})

		if err != nil {
			t.Fatalf("Error creating resource %s ::: failed with Error: %s", data.GetKind(), err)
		} else {
			// TODO
			dynamicClient.Invokes(te.NewCreateAction(gvr, data.GetNamespace(), data), data)
			resources = append(resources, data)
		}
	}
}

func run(inform *GenericInformer, stopper chan struct{}) {

}

func TestNewInformerWatcher(t *testing.T) {
	initAPIResources(t)
	stoppers := make(map[schema.GroupVersionResource]chan struct{})

	wg := sync.WaitGroup{}
	go func() {
		for {
			if gvrList != nil {
				// Create Informers for each test resource
				for _, gvr := range gvrList {
					t.Logf("Found resource %s, creating informer", gvr.String())

					// Create informer for each GroupVersionResource
					informer, _ := InformerForResource(gvr)
					t.Logf("Created %s informer %+v\n\n", gvr.Resource, informer)

					// In the test, we can simulate that the informer stopped. Allowing us to test the retry logic.
					stopper := make(chan struct{})
					stoppers[gvr] = stopper
				}
				t.Log("Total test informers running: ", len(stoppers))
			}
			// Breaking for test purposes.
			break
		}
		// After we handle every event and finish with the watcher and informer, we can exit out the test.
		wg.Done()
	}()
	// Similarly to how we keep the transformer routines running, we'll keep the test informers running.
	// However, after the test conditions we can decrement the waitgroup and exit the test.
	wg.Add(1)
	wg.Wait()
}
