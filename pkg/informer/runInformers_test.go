// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

// func fakeDiscoveryClient() *fake.FakeDiscovery {
// 	client := fakeclientset.NewSimpleClientset()
// 	fakeDiscovery := client.Discovery().(*fake.FakeDiscovery)

// 	// fakeDiscovery.Fake.Resources = fakeclientset.

// 	resources, _ := fakeDiscovery.ServerPreferredResources()
// 	fmt.Printf("fake resources: %+v\n", resources)

// 	return fakeDiscovery

// }

func fakeDiscoveryClient2() discovery.DiscoveryClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
		case "/api":
			obj = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/apis":
			obj = &metav1.APIGroupList{
				Groups: []metav1.APIGroup{
					{
						Name: "extensions",
						Versions: []metav1.GroupVersionForDiscovery{
							{GroupVersion: "extensions/v1beta1"},
						},
					},
				},
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(obj)
		if err != nil {
			// t.Fatalf("unexpected encoding error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))
	defer server.Close()
	client := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	resources, _ := client.ServerPreferredResources()
	fmt.Printf("\n>>>fake resources: %+v\n", resources)

	return *client //.(*discovery.DiscoveryInterface)
}

func Test_syncInformers(t *testing.T) {

	mockStoppers := make(map[schema.GroupVersionResource]chan struct{})
	mockAddFn := func(s string) func(interface{}) {
		return func(o interface{}) {
			t.Log("Procesing add...")
		}
	}
	mockUpdateFn := func(s string) func(interface{}, interface{}) {
		return func(old interface{}, new interface{}) {
			t.Log("Procesing update...")
		}
	}
	mockDeleteHandler := func(obj interface{}) {
		t.Log("Procesing delete...")
	}
	fakeClient := fakeDiscoveryClient2()

	syncInformers(fakeClient, mockStoppers, mockAddFn, mockUpdateFn, mockDeleteHandler)
}
