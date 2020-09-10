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
	"time"

	tr "github.com/open-cluster-management/search-collector/pkg/transforms"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

var dynamicClient = fake.FakeDynamicClient{}                  // Fake Dynamic client that the informer will be accessing.
var gvrList = make(map[string]schema.GroupVersionResource)    // GroupVersionResource list.
var resources = make(map[string][]*unstructured.Unstructured) // Key: GVR resource - Value: Test data for that resource.
var wg = sync.WaitGroup{}                                     // Wait group to monitor the gorountines being created.

// var input = make(chan *tr.Event)
// var output = make(chan tr.NodeEvent)

// The API Resource List is retreived through the discovery client
// so since we're skipping the discovery client and just using the GVR List, we can bypass using the API List
// var apiResourceList = []v1.APIResource{}

// We need the upsert transformer to send the data along the channels for the informer.
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

		// Set GVR resource to current data resource.
		var gvr schema.GroupVersionResource
		gvr.Resource = str.Join([]string{str.ToLower(kind), "s"}, "")

		// Set the GVR Group and Version
		apiVersion := str.Split(data.GetAPIVersion(), "/")

		if len(apiVersion) == 2 {
			gvr.Group = apiVersion[0]
			gvr.Version = apiVersion[1]
		} else {
			gvr.Version = apiVersion[0]
		}
		gvrList[gvr.Resource] = gvr

		// We need to create resources for the dynamic client.
		_, err := dynamicClient.Resource(gvr).Create(data, v1.CreateOptions{})

		if err != nil {
			t.Fatalf("Error creating resource %s ::: failed with Error: %s", data.GetKind(), err)
		} else {
			// TODO: Find a more efficient way to handle resource creation for the fake client.
			// The dynamic client is not setting the resource, so we cannot list the data from that client.
			// Therefore, for now, we'll create a map and store the resource data under the GVR resource key.
			resources[gvr.Resource] = append(resources[gvr.Resource], data)
		}
	}
}

// FakeRun simulate the informer run process.
func FakeRun(t *testing.T, inform *GenericInformer, stopper chan struct{}) {
	for {
		// Similarly to how we're using a retry logic in the informer, we need to test that the informer can be restarted.
		if inform.retries > 0 {
			wait := time.Duration(2 * time.Second)
			t.Logf("Preparing to wait %s before simulating testing retry", wait)
			time.Sleep(wait)
		}

		t.Log("(Re)starting informer with fake client: ", inform.gvr.String())
		FakeListAndResync(t, inform)  // List and Resync
		FakeWatch(t, inform, stopper) // Watcher

		if inform.stopped {
			break
		}
	}
	// Since informer has stopped, we can reduce the wg count.
	t.Log("Informer was stopped. ", inform.gvr.String())
}

// Create a fake object just with the UID
func fakeNewUnstructured(t *testing.T, uid string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"uid": uid,
			},
		},
	}
}

// FakeListAndResync implements a fake list and resync functionality, to mock the list and resync of the generic informer.
func FakeListAndResync(t *testing.T, inform *GenericInformer) {
	// We already stored the resources data for each informer within a map. So we can just access that within this function.

	// In order to test tthe retry logic, we need to set a previous list of resources.
	var prevResourceIndex map[string]string
	if len(inform.resourceIndex) > 0 {
		assert.Greater(t, inform.retries, int64(0)) // Test that this is the retry.
		prevResourceIndex = inform.resourceIndex
		inform.resourceIndex = make(map[string]string)
	}

	// Add all resources.
	for i := range resources[inform.gvr.Resource] {
		t.Logf("KIND: %s UUID: %s, ResourceVersion: %s",
			inform.gvr.Resource, resources[inform.gvr.Resource][i].GetUID(), resources[inform.gvr.Resource][i].GetResourceVersion())
		// TODO: Set up adding resources to event channel.
		inform.AddFunc(resources[inform.gvr.Resource][i])
		inform.resourceIndex[string(resources[inform.gvr.Resource][i].GetUID())] = resources[inform.gvr.Resource][i].GetResourceVersion()
	}

	// Test to see if the previous resource index is equal to the current resource index.
	if inform.retries > 0 {
		assert.Equal(t, inform.resourceIndex, prevResourceIndex)
	}

	t.Logf("Listed\t[Group: %s \tKind: %s]  ===>  resourceTotal: %d  resourceVersion: %s",
		inform.gvr.Group, inform.gvr.Resource, len(resources[inform.gvr.Resource]), resources[inform.gvr.Resource][0].GetResourceVersion())

	// Similiary to what we do in the informer list and resync func, we need to simulate a retry for testing purposes.
	for key := range prevResourceIndex {
		if _, exist := inform.resourceIndex[key]; !exist {
			t.Logf("Test resource does not exist. Deleting resource: %s with UID: %s", inform.gvr.Resource, key)
			obj := fakeNewUnstructured(t, key)
			inform.DeleteFunc(obj)
		}
	}
}

// FakeWatch implements a fake watcher for the informer.
func FakeWatch(t *testing.T, inform *GenericInformer, stopper chan struct{}) {
	t.Logf("Fake watching\t[Group: %s \tKind: %s]", inform.gvr.Group, inform.gvr.Resource)

	// We can simulate the retry logic first
	if inform.retries == 0 {
		t.Logf("Error watching resources for %s.", inform.gvr.String())
		inform.retries++
		return
	}

	// Create watch events
	// watchEvents := make(<-chan Event)

	// Reset informer retry
	inform.retries = 0

	// TODO: Incoporate events for the informer to watch and process.
	// Since we're already adding data in the fake list and resync
	// we can test updating and deleting the gvr resource.

	// t.Logf("Received MODIFY event. Kind: %s ", inform.gvr.Resource)
	// t.Logf("Received DELETE event. Kind: %s ", inform.gvr.Resource)

	// Since we're simulating the retry logic, we can stop the informer after it has tested for the events.
	inform.stopped = true
	wg.Done()
}

func TestNewInformerWatcher(t *testing.T) {
	initAPIResources(t)
	stoppers := make(map[schema.GroupVersionResource]chan struct{})

	createFakeInformerAddHandler := func(resourceName string) func(interface{}) {
		return func(obj interface{}) {
			res := obj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Create,
				Resource:       res,
				ResourceString: resourceName,
			}
			t.Log("AddFunc", upsert)
		}
	}

	createFakeInformerUpdateHandler := func(resourceName string) func(interface{}, interface{}) {
		return func(oldObj, newObj interface{}) {
			res := newObj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Update,
				Resource:       res,
				ResourceString: resourceName,
			}
			t.Log("UpdateFunc", upsert)
		}
	}

	fakeInformerDeleteHandler := func(obj interface{}) {
		res := obj.(*unstructured.Unstructured)
		upsert := tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Delete,
			Node: tr.Node{
				UID: str.Join([]string{"local-cluster", string(res.GetUID())}, "/"),
			},
		}
		t.Log("DeleteFunc", upsert)
	}

	go func() {
		for {
			if gvrList != nil {
				// Create Informers for each test resource
				// TODO: Implement stoppers

				for _, gvr := range gvrList {
					t.Logf("Found resource %s, creating informer", gvr.String())

					// Create informer for each GroupVersionResource
					informer, _ := InformerForResource(gvr)
					t.Logf("Created %s informer", gvr.Resource)

					// Set the fake informer functions
					informer.AddFunc = createFakeInformerAddHandler(gvr.Resource)
					informer.UpdateFunc = createFakeInformerUpdateHandler(gvr.Resource)
					informer.DeleteFunc = fakeInformerDeleteHandler

					// In the test, we can simulate that the informer stopped. Allowing us to test the retry logic.
					stopper := make(chan struct{})
					stoppers[gvr] = stopper
					go FakeRun(t, &informer, stopper)
					wg.Add(1)
				}
				t.Log("Total test informers running: ", len(stoppers))
			}

			// Breaking for dev/test purposes.
			wg.Done()
			if len(stoppers) == len(gvrList) {
				break
			}
		}
		// After we handle every event and finish with the watcher and informer, we can exit out the test.
	}()
	// Similarly to how we keep the transformer routines running, we'll keep the test informers running.
	// However, after the test conditions we can decrement the waitgroup and exit the test.
	wg.Add(1)
	wg.Wait()
}
