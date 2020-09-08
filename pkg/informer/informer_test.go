/*
Copyright (c) 2020 Red Hat, Inc.
*/
package informer

import (
	"encoding/json"
	"io/ioutil"
	"log"
	str "strings"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

var dynamicClient fake.FakeDynamicClient
var apiResourceList []v1.APIResource
var gvrList []schema.GroupVersionResource

func initAPIResources(t *testing.T) {
	dir := "../../test-data"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	dynamicClient = fake.FakeDynamicClient{}
	apiResourceList = []v1.APIResource{}
	gvrList = []schema.GroupVersionResource{}

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

		newResource := v1.APIResource{}
		newResource.Name = data.GetName()
		newResource.Kind = kind

		apiVersion := str.Split(data.GetAPIVersion(), "/")

		var gvr schema.GroupVersionResource

		// Set GVR resource to current data resource.
		gvr.Resource = str.Join([]string{str.ToLower(kind), "s"}, "")

		// Set the GVR Group and Version
		if len(apiVersion) == 2 {
			newResource.Group = apiVersion[0]
			newResource.Version = apiVersion[1]

			gvr.Group = newResource.Group
			gvr.Version = newResource.Version
		} else {
			newResource.Version = apiVersion[0]
			gvr.Version = newResource.Version
		}

		gvrList = append(gvrList, gvr)

		// Not all resources will be namespaced, so we need to check for that value.
		if data.GetNamespace() != "" {
			newResource.Namespaced = true
		}

		t.Logf("[newResource]\t ===> %+v\n\n", newResource)
		apiResourceList = append(apiResourceList, newResource)

		// We need to create resources for the dynamic client.
		_, err := dynamicClient.Resource(gvr).Create(data, v1.CreateOptions{})

		if err != nil {
			t.Fatalf("Error creating resource %s ::: failed with Error: %s", data.GetKind(), err)
		}
	}
}

func TestNewInformerWatcher(t *testing.T) {
	initAPIResources(t)

	for _, gvr := range gvrList {
		t.Log(dynamicClient.Resource(gvr))
	}

	// TODO: It would be neat to use a fake discovery client to discovery fake resources
	// simliar to how we're using it in main to get the gvr
	// var discoveryClient fake.FakeDiscovery

	//  // TODO: It would be useful for us to initialize the data similiarliy to how we are setting the data in main.
	//  // However the dynamicClient that we're using needs a kubeConfig, so we don't actually have resources to list.
	//  // Create fake data for the dynamic resource.

	// fmt.Println("dynamic", dynamicClient)
	// fmt.Println("resources", apiResourceList)

	// dynamicClient.Resources = []*v1.APIResourceList{
	//  &v1.APIResourceList: {
	//      APIResources: apiResourceList,
	//  },
	// }
	// for i, resource := range dynamicClient.Resources {
	//  InformerForResource(resource.g)
	// }
	// genInformer := GenericInformer{}
	// gen := &genInformer
	// listAndResync(gen, dynamicClient)
}

/*func TestGetSelectedPods(t *testing.T) {
    t.Parallel()
    data := []struct {
        clientset         kubernetes.Interface
        countExpectedPods int
        inputNamespace    string
        listOpt           v1.ListOptions
        err               error
    }{
        // Pods are in the system but they do not match the creteria
        {
            clientset: fake.NewSimpleClientset(&v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:        "influxdb-v2",
                    Namespace:   "default",
                    Annotations: map[string]string{},
                },
            }, &v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:        "chronograf",
                    Namespace:   "default",
                    Annotations: map[string]string{},
                },
            }),
            inputNamespace:    "default",
            countExpectedPods: 0,
        },
        // there are not pods in the default namespace with the right annotation and in status running
        {
            clientset: fake.NewSimpleClientset(&v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:      "influxdb-v2",
                    Namespace: "hola",
                    Annotations: map[string]string{
                        ProfefeEnabledAnnotation: "true",
                    },
                },
                Status: v1.PodStatus{
                    Phase: v1.PodRunning,
                },
            }, &v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:        "chronograf",
                    Namespace:   "none",
                    Annotations: map[string]string{},
                },
            }),
            inputNamespace:    "default",
            countExpectedPods: 0,
        },
        // there is a pod in the default namespace with the right annotation and in status running
        {
            clientset: fake.NewSimpleClientset(&v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:      "influxdb-v2",
                    Namespace: "default",
                    Annotations: map[string]string{
                        ProfefeEnabledAnnotation: "true",
                    },
                },
                Status: v1.PodStatus{
                    Phase: v1.PodRunning,
                },
            }, &v1.Pod{
                ObjectMeta: v1.ObjectMeta{
                    Name:        "chronograf",
                    Namespace:   "none",
                    Annotations: map[string]string{},
                },
            }),
            inputNamespace:    "default",
            countExpectedPods: 1,
        },
    }
    for _, single := range data {
        t.Run("", func(single struct {
            clientset         kubernetes.Interface
            countExpectedPods int
            inputNamespace    string
            listOpt           v1.ListOptions
            err               error
        }) func(t *testing.T) {
            return func(t *testing.T) {
                pods, err := GetSelectedPods(single.clientset, single.inputNamespace, single.listOpt)
                if err != nil {
                    if single.err == nil {
                        t.Fatalf(err.Error())
                    }
                    if !strings.EqualFold(single.err.Error(), err.Error()) {
                        t.Fatalf("expected err: %s got err: %s", single.err, err)
                    }
                } else {
                    if len(pods) != single.countExpectedPods {
                        t.Fatalf("expected %d pods, got %d", single.countExpectedPods, len(pods))
                    }
                }
            }
        }(single))
    }
}*/
