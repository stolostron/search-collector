// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stolostron/search-collector/pkg/config"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

var mockAddFn = func(gvr schema.GroupVersionResource) func(interface{}) {
	return func(o interface{}) {}
}

var mockUpdateFn = func(gvr schema.GroupVersionResource) func(interface{}, interface{}) {
	return func(old interface{}, new interface{}) {}
}

var mockDeleteHandler = func(obj interface{}) {}

func fakeDiscoveryClient() (*httptest.Server, discovery.DiscoveryClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
		case "/api", "/apis":
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
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, writeErr := w.Write(output)
		if writeErr != nil {
			return
		}
	}))
	client := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	return server, *client
}

func Test_syncInformers(t *testing.T) {
	// Establish the config
	config.InitConfig()

	registry := make(map[schema.GroupVersionResource]informerEntry)

	fakeServer, fakeClient := fakeDiscoveryClient()
	defer fakeServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncInformers(ctx, fakeClient, registry, mockAddFn, mockUpdateFn, mockDeleteHandler)

	assert.Equal(t, 3, len(registry))

	podEntry, exists := registry[schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}]
	assert.True(t, exists)
	assert.NotNil(t, podEntry.cancel)
}

// Validate that informer is stopped when resource no longer exists.
func Test_syncInformers_removeInformers(t *testing.T) {
	registry := make(map[schema.GroupVersionResource]informerEntry)
	ctx, cancel := context.WithCancel(context.Background())
	registry[schema.GroupVersionResource{Group: "", Version: "v1", Resource: "notExist"}] = informerEntry{cancel: cancel}

	fakeServer, fakeClient := fakeDiscoveryClient()
	defer fakeServer.Close()

	syncInformers(ctx, fakeClient, registry, mockAddFn, mockUpdateFn, mockDeleteHandler)

	assert.Equal(t, 3, len(registry))

	// Validate that informer is stopped when resource no longer exists.
	_, exists := registry[schema.GroupVersionResource{Group: "", Version: "v1", Resource: "notExist"}]
	assert.False(t, exists)
}

func getSimpleTransformedCRD() unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name":            "fakes.policy.open-cluster-management.io",
				"generation":      int64(1),
				"resourceVersion": "1",
			},
			"spec": map[string]interface{}{
				"group": "policy.open-cluster-management.io",
				"versions": []interface{}{
					map[string]interface{}{
						"additionalPrinterColumns": []interface{}{
							map[string]interface{}{
								"jsonPath": ".status.compliant",
								"name":     "Compliance state",
								"type":     "string",
							},
						},
						"name":    "v1",
						"storage": true,
					},
				},
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"reason": "InitialNamesAccepted",
						"status": "True",
						"type":   "Established",
					},
				},
			},
		},
	}
}

func Test_isCRDEstablished(t *testing.T) {
	crd := unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{},
		},
	}
	assert.False(t, isCRDEstablished(&crd))

	crd = unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"reason": "InitialNamesAccepted",
						"status": "False",
						"type":   "Established",
					},
				},
			},
		},
	}
	assert.False(t, isCRDEstablished(&crd))

	crd = unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"reason": "NoConflicts",
						"status": "True",
						"type":   "NamesAccepted",
					},
					map[string]interface{}{
						"reason": "InitialNamesAccepted",
						"status": "True",
						"type":   "Established",
					},
				},
			},
		},
	}
	assert.True(t, isCRDEstablished(&crd))
}

func Test_transformCRD(t *testing.T) {
	crd := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name":            "fakes.policy.open-cluster-management.io",
				"generation":      int64(1),
				"resourceVersion": "1",
				"labels": map[string]interface{}{
					"hello": "world",
				},
			},
			"spec": map[string]interface{}{
				"group": "policy.open-cluster-management.io",
				"versions": []interface{}{
					map[string]interface{}{
						"name":    "v1beta1",
						"storage": false,
						"served":  true,
					},
					map[string]interface{}{
						"additionalPrinterColumns": []interface{}{
							map[string]interface{}{
								"jsonPath": ".status.compliant",
								"name":     "Compliance state",
								"type":     "string",
							},
						},
						"name":    "v1",
						"storage": true,
						"served":  true,
					},
				},
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"reason": "InitialNamesAccepted",
						"status": "True",
						"type":   "Established",
					},
				},
			},
		},
	}

	expectedCRD := getSimpleTransformedCRD()

	transformedCRD, err := transformCRD(&crd)
	assert.Nil(t, err)
	assert.Equal(t, expectedCRD.Object, transformedCRD.(*unstructured.Unstructured).Object)
}

func Test_gvrFromCRD(t *testing.T) {
	crd := getSimpleTransformedCRD()

	gvr, err := gvrFromCRD(&crd)
	assert.Nil(t, err)

	expected := schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "fakes",
	}

	assert.Equal(t, expected, *gvr)
}

func Test_toLowerCamelCase(t *testing.T) {
	assert.Equal(t, "enforcementAction", toLowerCamelCase("enforcement action"))
	assert.Equal(t, "enforcementAction", toLowerCamelCase("enforcement-action"))
	assert.Equal(t, "enforcementAction", toLowerCamelCase("enforcementAction"))
}

func Test_gvrToPrinterColumns(t *testing.T) {
	gvrToColumns := gvrToPrinterColumns{mapping: map[schema.GroupVersionResource][]tr.ExtractProperty{}}

	gvr := schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "fakes",
	}

	// At first, the mapping should be empty.
	assert.Nil(t, gvrToColumns.get(gvr))

	crd := newSimpleCRDWithAdditionalPrinterColumns()

	// Cache the mapping for the CRD
	assert.Nil(t, gvrToColumns.set(&crd))

	zerVal := 0
	expected := []tr.ExtractProperty{
		{Name: "complianceState", JSONPath: "{.status.compliant}", Priority: &zerVal},
	}

	// Verify the correct GVR was parsed and the mapping was stored
	assert.Equal(t, expected, gvrToColumns.get(gvr))

	// Verify the mapping can be unset
	assert.Nil(t, gvrToColumns.unset(&crd))
	assert.Nil(t, gvrToColumns.get(gvr))
}

func Test_gvrToPrinterColumnsWithPriority(t *testing.T) {
	gvrToColumns := gvrToPrinterColumns{mapping: map[schema.GroupVersionResource][]tr.ExtractProperty{}}

	gvr := schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "fakes",
	}

	crd := newSimpleCRDWithAdditionalPrinterColumns()

	// Set a non-zero priority on the printer column.
	versions := crd.Object["spec"].(map[string]interface{})["versions"].([]interface{})
	storageVersion := versions[1].(map[string]interface{})
	columns := storageVersion["additionalPrinterColumns"].([]interface{})
	columns[0].(map[string]interface{})["priority"] = int64(1)

	assert.Nil(t, gvrToColumns.set(&crd))

	priority := 1
	expected := []tr.ExtractProperty{
		{Name: "complianceState", JSONPath: "{.status.compliant}", Priority: &priority},
	}

	assert.Equal(t, expected, gvrToColumns.get(gvr))
}

func newSimpleCRDWithAdditionalPrinterColumns() unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name":            "fakes.policy.open-cluster-management.io",
				"generation":      int64(1),
				"resourceVersion": "1",
				"labels": map[string]interface{}{
					"hello": "world",
				},
			},
			"spec": map[string]interface{}{
				"group": "policy.open-cluster-management.io",
				"versions": []interface{}{
					map[string]interface{}{
						"name":    "v1beta1",
						"storage": false,
						"served":  true,
					},
					map[string]interface{}{
						"additionalPrinterColumns": []interface{}{
							map[string]interface{}{
								"jsonPath": ".status.compliant",
								"name":     "Compliance state",
								"type":     "string",
							},
						},
						"name":    "v1",
						"storage": true,
						"served":  true,
					},
				},
			},
		},
	}
}

func TestGroupFromConfigKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Pod", ""},
		{"Deployment.apps", "apps"},
		{"*.apps", "apps"},
		{"*", ""},
		{"Policy.policy.open-cluster-management.io", "policy.open-cluster-management.io"},
		{"*.monitoring.coreos.com", "monitoring.coreos.com"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, groupFromConfigKey(tc.input))
		})
	}
}

func TestTriggerResyncForConfigKeys(t *testing.T) {
	// Drain any existing items from the channel.
	select {
	case <-resyncQueue:
	default:
	}

	keys := []string{"Pod", "Deployment.apps"}
	TriggerResyncForConfigKeys(keys)

	received := <-resyncQueue
	assert.Equal(t, keys, received)
}

func TestTriggerResyncForConfigKeys_Coalesce(t *testing.T) {
	// Drain any existing items from the channel.
	select {
	case <-resyncQueue:
	default:
	}

	first := []string{"Pod"}
	second := []string{"Secret", "Node"}

	TriggerResyncForConfigKeys(first)
	TriggerResyncForConfigKeys(second) // should be dropped (non-blocking, cap 1)

	received := <-resyncQueue
	assert.Equal(t, first, received, "only the first batch should be in the queue")

	// Channel should be empty now.
	select {
	case extra := <-resyncQueue:
		t.Errorf("expected empty queue, got %v", extra)
	default:
		// expected
	}
}
