// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

func fakeDiscoveryClient() (*httptest.Server, discovery.DiscoveryClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
		case "/api":
			obj = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/api/v1":
			obj = metav1.APIResourceList{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{Name: "pods", Namespaced: true, Kind: "Pod", Verbs: []string{"list", "watch"}},
					{Name: "services", Namespaced: true, Kind: "Service", Verbs: []string{"list", "watch"}},
					{Name: "namespaces", Namespaced: false, Kind: "Namespace", Verbs: []string{"list", "watch"}},
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
		_, writeErr := w.Write(output)
		if writeErr != nil {
			fmt.Println("error", err)
		}
	}))
	client := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	return server, *client
}

func Test_syncInformers(t *testing.T) {

	mockStoppers := make(map[schema.GroupVersionResource]chan struct{})
	mockAddFn := func(s string) func(interface{}) {
		return func(o interface{}) {}
	}
	mockUpdateFn := func(s string) func(interface{}, interface{}) {
		return func(old interface{}, new interface{}) {}
	}
	mockDeleteHandler := func(obj interface{}) {}
	fakeServer, fakeClient := fakeDiscoveryClient()
	defer fakeServer.Close()

	syncInformers(fakeClient, mockStoppers, mockAddFn, mockUpdateFn, mockDeleteHandler)

	assert.Equal(t, 3, len(mockStoppers))
}
