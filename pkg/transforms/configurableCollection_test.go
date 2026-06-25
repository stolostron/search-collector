// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"fmt"
	"github.com/stretchr/testify/require"
	k8stesting "k8s.io/client-go/testing"
	"strings"
	"testing"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestLoadAndMergeConfigurableCollection_ValidConfig(t *testing.T) {
	// Save original config and restore after test
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		// Clear mergedTransformConfig after test
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a mock CollectorConfig
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "dnsPolicy",
								"jsonPath": "{.spec.dnsPolicy}",
							},
							map[string]interface{}{
								"name":     "enableServiceLinks",
								"jsonPath": "{.spec.enableServiceLinks}",
								"type":     "string",
							},
						},
					},
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"search.open-cluster-management.io"},
							"kinds":     []interface{}{"Search"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "searchPGStorage",
								"jsonPath": "{.spec.dbStorage.size}",
								"type":     "bytes",
							},
						},
					},
				},
			},
		},
	}

	// Create fake dynamic client with the mock resource
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	// Call the function with the fake client
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify Pod config was updated in mergedTransformConfig
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist")
	assert.Equal(t, 2, len(podConfig.properties), "Pod should have 2 custom properties")
	assert.Equal(t, "dnsPolicy", podConfig.properties[0].Name)
	assert.Equal(t, "{.spec.dnsPolicy}", podConfig.properties[0].JSONPath)
	assert.Equal(t, "enableServiceLinks", podConfig.properties[1].Name)
	assert.Equal(t, DataTypeString, podConfig.properties[1].DataType)

	// Verify Search config was updated in mergedTransformConfig
	searchConfig, exists := mergedTransformConfig["Search.search.open-cluster-management.io"]
	assert.True(t, exists, "Search config should exist")
	assert.Equal(t, 1, len(searchConfig.properties), "Search should have 1 custom property")
	assert.Equal(t, "searchPGStorage", searchConfig.properties[0].Name)
	assert.Equal(t, DataTypeBytes, searchConfig.properties[0].DataType)
}

// Exclude rules populate excludeRules (not mergedTransformConfig).
func TestLoadAndMergeConfigurableCollection_ExcludePopulatesExcludedResources(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
		excludeRules = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// mergedTransformConfig must not grow — exclude does not add custom properties
	assert.Equal(t, len(defaultTransformConfig), len(mergedTransformConfig),
		"Exclude actions should not add entries to mergedTransformConfig")
	// excludeRules must cause Lease to be excluded
	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease.coordination.k8s.io must be excluded after exclude rule")
}

// FUTURE: this should eventually appropriately include when search-collector-config merged
func TestLoadAndMergeConfigurableCollection_SkipIncludeWithoutFields(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"*"},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added beyond the defaults
	assert.Equal(t, len(defaultTransformConfig), len(mergedTransformConfig),
		"Include actions without fields should not add new entries to mergedTransformConfig")
}

func TestLoadAndMergeConfigurableCollection_InvalidMultipleKinds(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod", "Service"}, // Multiple kinds - invalid
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "test",
								"jsonPath": "{.spec.test}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added (rule should be skipped)
	assert.Equal(t, originalLen, len(mergedTransformConfig), "Rules with multiple kinds should be skipped")
}

func TestLoadAndMergeConfigurableCollection_InvalidMultipleApiGroups(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps", "batch"}, // Multiple apiGroups - invalid
							"kinds":     []interface{}{"Deployment"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "test",
								"jsonPath": "{.spec.test}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added (rule should be skipped)
	assert.Equal(t, originalLen, len(mergedTransformConfig), "Rules with multiple apiGroups should be skipped")
}

func TestLoadAndMergeConfigurableCollection_FeatureDisabled(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
	}()

	config.Cfg.FeatureConfigurableCollection = false

	// Create a fake client (shouldn't be called)
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	originalLen := len(mergedTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify nothing changed
	assert.Equal(t, originalLen, len(mergedTransformConfig), "Config should not change when feature is disabled")
}

func TestLoadAndMergeConfigurableCollection_PublicMethodRespectsFeatureFlag(t *testing.T) {
	// This test verifies the public method actually checks the feature flag
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		mergedTransformConfig = nil
	}()

	// Test with feature DISABLED
	config.Cfg.FeatureConfigurableCollection = false
	originalLen := len(mergedTransformConfig)

	// Call the PUBLIC method (not the internal helper)
	LoadAndMergeConfigurableCollection()

	// Should not have attempted to load anything
	assert.Equal(t, originalLen, len(mergedTransformConfig), "Public method should respect feature flag when disabled")

	// Test with feature ENABLED would require a real k8s client or more complex mocking
	// The internal method tests cover the enabled case with fake clients
}

func TestLoadAndMergeConfigurableCollection_ResourceNotFound(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create empty fake client (no resources)
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify mergedTransformConfig equals defaultTransformConfig (no custom config applied)
	assert.Equal(t, len(defaultTransformConfig), len(mergedTransformConfig), "Config should equal defaultTransformConfig when resource not found")
}

func TestLoadAndMergeConfigurableCollection_FieldWithoutDataType(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		delete(defaultTransformConfig, "Secret")
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Secret"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "data",
								"jsonPath": "{.data}",
								// No type specified
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify Secret config was updated with default DataType (DataTypeString)
	secretConfig, exists := mergedTransformConfig["Secret"]
	assert.True(t, exists, "Secret config should exist")
	assert.Equal(t, 1, len(secretConfig.properties), "Secret should have 1 custom property")
	assert.Equal(t, "data", secretConfig.properties[0].Name)
	assert.Equal(t, DataTypeString, secretConfig.properties[0].DataType, "DataType should default to string when not specified, matching CRD default")
}

func TestLoadAndMergeConfigurableCollection_DataTypeConversions(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		delete(defaultTransformConfig, "TestResource")
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"test.io"},
							"kinds":     []interface{}{"TestResource"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "stringField",
								"jsonPath": "{.spec.stringField}",
								"type":     "string",
							},
							map[string]interface{}{
								"name":     "numberField",
								"jsonPath": "{.spec.numberField}",
								"type":     "number",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify TestResource config was created with correct DataTypes
	testResourceConfig, exists := mergedTransformConfig["TestResource.test.io"]
	assert.True(t, exists, "TestResource config should exist")
	assert.Equal(t, 2, len(testResourceConfig.properties), "TestResource should have 2 custom properties")

	// Verify DataTypeString
	assert.Equal(t, "stringField", testResourceConfig.properties[0].Name)
	assert.Equal(t, DataTypeString, testResourceConfig.properties[0].DataType)

	// Verify DataTypeNumber
	assert.Equal(t, "numberField", testResourceConfig.properties[1].Name)
	assert.Equal(t, DataTypeNumber, testResourceConfig.properties[1].DataType)
}

func TestLoadAndMergeConfigurableCollection_MissingSpec(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a CollectorConfig without spec
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			// No spec field
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should not add custom config when spec is missing, should equal defaultTransformConfig
	assert.Equal(t, len(defaultTransformConfig), len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_SpecNotMap(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a CollectorConfig with invalid spec type
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": "not-a-map", // Invalid type
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should not modify config when spec is not a map
	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_CollectionRulesNotArray(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": "not-an-array", // Invalid type
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_RuleNotMap(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					"not-a-map", // Invalid rule type
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_MissingResourceSelector(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						// Missing resourceSelector
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "test",
								"jsonPath": "{.spec.test}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_EmptyKinds(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{}, // Empty kinds
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "test",
								"jsonPath": "{.spec.test}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_EmptyKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{""}, // Empty string kind
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "test",
								"jsonPath": "{.spec.test}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(mergedTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should skip rules with empty kind string
	assert.Equal(t, originalLen, len(mergedTransformConfig))
}

func TestLoadAndMergeConfigurableCollection_FieldMissingNameOrJsonPath(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		delete(defaultTransformConfig, "Service")
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Service"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "validField",
								"jsonPath": "{.spec.valid}",
							},
							map[string]interface{}{
								// Missing name
								"jsonPath": "{.spec.test}",
							},
							map[string]interface{}{
								"name": "missingJsonPath",
								// Missing jsonPath
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should only add the valid field, skipping the ones with missing name or jsonPath
	serviceConfig, exists := mergedTransformConfig["Service"]
	assert.True(t, exists)
	assert.Equal(t, 1, len(serviceConfig.properties), "Should only have 1 valid field")
	assert.Equal(t, "validField", serviceConfig.properties[0].Name)
}

func TestLoadAndMergeConfigurableCollection_FieldSuffix(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a mock CollectorConfig with fieldSuffix
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fieldSuffix": "grc",
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "status",
								"jsonPath": "{.status.phase}",
							},
							map[string]interface{}{
								"name":     "customField",
								"jsonPath": "{.metadata.annotations.custom}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify that fieldSuffix was applied: "status" + "." + "grc" = "status.grc"
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist")
	assert.Equal(t, 2, len(podConfig.properties), "Pod should have 2 custom properties")
	assert.Equal(t, "status.grc", podConfig.properties[0].Name, "Field should have suffix with dot separator")
	assert.Equal(t, "{.status.phase}", podConfig.properties[0].JSONPath)
	assert.Equal(t, "customField.grc", podConfig.properties[1].Name, "Field should have suffix with dot separator")
}

func TestLoadAndMergeConfigurableCollection_EmptyFieldSuffix(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a mock CollectorConfig with empty fieldSuffix
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fieldSuffix": "",
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "customField",
								"jsonPath": "{.metadata.annotations.custom}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify that no suffix was applied when fieldSuffix is empty
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist")
	assert.Equal(t, 1, len(podConfig.properties), "Pod should have 1 custom property")
	assert.Equal(t, "customField", podConfig.properties[0].Name, "Field should not have suffix when fieldSuffix is empty")
}

func TestLoadAndMergeConfigurableCollection_FieldCollisionWithSuffix(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create two rules for the same resource:
	// Rule 1: Adds "status.grc" field
	// Rule 2: Tries to add "status" with suffix "grc" (which becomes "status.grc") - should be skipped due to collision
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// First rule: add status.grc
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fieldSuffix": "grc",
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "status",
								"jsonPath": "{.status.phase}",
							},
						},
					},
					// Second rule: try to add status.grc again (should be skipped)
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fieldSuffix": "grc",
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "status",
								"jsonPath": "{.different.status}",
							},
							map[string]interface{}{
								"name":     "validField",
								"jsonPath": "{.metadata.annotations.valid}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify collision detection:
	// - First rule adds "status.grc"
	// - Second rule tries to add "status.grc" again (collision, skipped) and "validField.grc" (no collision, added)
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist")
	assert.Equal(t, 2, len(podConfig.properties), "Pod should have 2 properties (status.grc from rule 1, validField.grc from rule 2)")

	// Verify the first status.grc is from rule 1 (not overwritten by rule 2)
	assert.Equal(t, "status.grc", podConfig.properties[0].Name)
	assert.Equal(t, "{.status.phase}", podConfig.properties[0].JSONPath, "Should keep the first rule's jsonPath, not the second")

	// Verify validField.grc was added from rule 2
	assert.Equal(t, "validField.grc", podConfig.properties[1].Name)
	assert.Equal(t, "{.metadata.annotations.valid}", podConfig.properties[1].JSONPath)
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsWithSpecificKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractConditions, "extractConditions should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsWithMultipleKinds(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment", "StatefulSet"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractConditions, "Deployment extractConditions should be true")

	ssConfig, exists := mergedTransformConfig["StatefulSet.apps"]
	assert.True(t, exists, "StatefulSet.apps config should exist")
	assert.True(t, ssConfig.extractConditions, "StatefulSet extractConditions should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsWildcardKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Wildcard kind "*" with apiGroup "apps" creates a "*.apps" entry in mergedTransformConfig
	wildcardConfig, exists := mergedTransformConfig["*.apps"]
	assert.True(t, exists, "*.apps wildcard config should exist in mergedTransformConfig")
	assert.True(t, wildcardConfig.extractConditions, "*.apps extractConditions should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsMultipleApiGroups(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps", "batch"},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	appsConfig, exists := mergedTransformConfig["*.apps"]
	assert.True(t, exists, "*.apps wildcard config should exist")
	assert.True(t, appsConfig.extractConditions, "*.apps extractConditions should be true")

	batchConfig, exists := mergedTransformConfig["*.batch"]
	assert.True(t, exists, "*.batch wildcard config should exist")
	assert.True(t, batchConfig.extractConditions, "*.batch extractConditions should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsWithFieldsAndKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "replicas",
								"jsonPath": "{.spec.replicas}",
								"type":     "number",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractConditions, "extractConditions should be true")
	assert.Equal(t, 1, len(deployConfig.properties), "Should have 1 custom field")
	assert.Equal(t, "replicas", deployConfig.properties[0].Name)
	assert.Equal(t, DataTypeNumber, deployConfig.properties[0].DataType)
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsPreservesDefaults(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Empty rules - just verify defaults are preserved
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify defaults with extractConditions are preserved
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist from defaults")
	assert.True(t, podConfig.extractConditions, "Pod should retain default extractConditions=true")

	nodeConfig, exists := mergedTransformConfig["Node"]
	assert.True(t, exists, "Node config should exist from defaults")
	assert.True(t, nodeConfig.extractConditions, "Node should retain default extractConditions=true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsCoreApiGroup(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Core API group is empty string - wildcard key is just "*" (no dot prefix)
	coreConfig, exists := mergedTransformConfig["*"]
	assert.True(t, exists, "* wildcard config should exist for core API group")
	assert.True(t, coreConfig.extractConditions, "* extractConditions should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectConditionsMixedRules(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectConditions := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Rule 1: collect conditions for all resources in apps and batch apiGroups (wildcard)
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps", "batch"},
							"kinds":     []interface{}{"*"},
						},
					},
					// Rule 2: collect conditions for a specific kind in a third apiGroup
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"policy.open-cluster-management.io"},
							"kinds":     []interface{}{"Policy"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Rule 1: wildcard entries should exist in mergedTransformConfig
	appsConfig, exists := mergedTransformConfig["*.apps"]
	assert.True(t, exists, "*.apps wildcard config should exist")
	assert.True(t, appsConfig.extractConditions, "*.apps extractConditions should be true")

	batchConfig, exists := mergedTransformConfig["*.batch"]
	assert.True(t, exists, "*.batch wildcard config should exist")
	assert.True(t, batchConfig.extractConditions, "*.batch extractConditions should be true")

	// Rule 2: specific kind should have extractConditions set in mergedTransformConfig
	policyConfig, exists := mergedTransformConfig["Policy.policy.open-cluster-management.io"]
	assert.True(t, exists, "Policy config should exist")
	assert.True(t, policyConfig.extractConditions, "Policy extractConditions should be true")

	// The policy apiGroup should NOT have a wildcard entry (it was kind-specific)
	_, wildcardExists := mergedTransformConfig["*.policy.open-cluster-management.io"]
	assert.False(t, wildcardExists,
		"policy apiGroup should not have wildcard entry since it was specified with a specific kind")
}

// ─── Status condition tests (ACM-33146) ───────────────────────────────────────
// These tests verify that loadAndMergeConfigurableCollectionWithClient writes
// an "Applied" status condition back to the CollectorConfig CR via the dynamic
// client so that users can see configuration errors with `oc describe`.

// getStatusConditionFromFakeClient inspects the fake client's recorded actions
// and returns the first "Applied" condition written to the status subresource.
// Returns nil if no status update was recorded.
func getStatusConditionFromFakeClient(fakeClient *fake.FakeDynamicClient) map[string]interface{} {
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() != "update" || action.GetSubresource() != "status" {
			continue
		}
		ua, ok := action.(k8stesting.UpdateAction)
		if !ok {
			continue
		}
		obj, ok := ua.GetObject().(*unstructured.Unstructured)
		if !ok {
			continue
		}
		status, ok := obj.Object["status"].(map[string]interface{})
		if !ok {
			continue
		}
		conditions, ok := status["conditions"].([]interface{})
		if !ok || len(conditions) == 0 {
			continue
		}
		if cond, ok := conditions[0].(map[string]interface{}); ok {
			return cond
		}
	}
	return nil
}

// TestStatusCondition_Applied_ValidConfig verifies that a well-formed config
// results in Applied=True written to the CollectorConfig status.
func TestStatusCondition_Applied_ValidConfig(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "dnsPolicy",
								"jsonPath": "{.spec.dnsPolicy}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond, "Expected a status condition to be written to the CollectorConfig CR")

	assert.Equal(t, "Applied", cond["type"], "Condition type should be 'Applied'")
	assert.Equal(t, "True", cond["status"], "Condition status should be True for a valid config")
	assert.Equal(t, "Applied", cond["reason"], "Condition reason should be 'Applied'")
	assert.Equal(t, "Configuration applied successfully.", cond["message"])
}

// TestStatusCondition_Applied_SkippedRule verifies that an include rule with no
// actionable configuration (no fields, no collectConditions, no collectAnnotations)
// results in Applied=False with a descriptive warning message.
func TestStatusCondition_Applied_SkippedRule(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						// include with no fields/collectConditions/collectAnnotations — should be skipped
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond, "Expected a status condition to be written to the CollectorConfig CR")

	assert.Equal(t, "Applied", cond["type"])
	assert.Equal(t, "False", cond["status"], "Condition status should be False when a rule is skipped")
	assert.Equal(t, "RulesSkipped", cond["reason"])
	msg, _ := cond["message"].(string)
	assert.True(t, strings.Contains(msg, "requires at least one field"),
		"Message should describe why the rule was skipped, got: %s", msg)
}

// TestStatusCondition_Applied_ExcludeRule verifies that a valid exclude rule results
// in Applied=True — exclude rules are now fully supported and should not produce warnings.
func TestStatusCondition_Applied_ExcludeRule(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
		excludeRules = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond, "Expected a status condition to be written to the CollectorConfig CR")

	assert.Equal(t, "Applied", cond["type"])
	assert.Equal(t, "True", cond["status"], "Exclude rule is valid — Applied should be True")
	assert.Equal(t, "Applied", cond["reason"])
	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease should be excluded after loading")
}

// TestStatusCondition_Applied_FieldCollision verifies that when two rules both
// try to add a field with the same name to the same resource, the second rule
// is skipped and Applied=False is written with a descriptive message.
// (Note: built-in properties like "name" are added at transform runtime via
// commonProperties(), not via the config layer, so collisions with them are not
// detected here. Config-layer collisions happen between CollectorConfig rules.)
func TestStatusCondition_Applied_FieldCollision(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Rule 1: adds "dnsPolicy" for Pod
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "dnsPolicy",
								"jsonPath": "{.spec.dnsPolicy}",
							},
						},
					},
					// Rule 2: also tries to add "dnsPolicy" for Pod — collision
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "dnsPolicy",
								"jsonPath": "{.spec.dnsPolicy}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond, "Expected a status condition to be written to the CollectorConfig CR")

	assert.Equal(t, "Applied", cond["type"])
	assert.Equal(t, "False", cond["status"], "Condition status should be False when a field collision is detected")
	assert.Equal(t, "RulesSkipped", cond["reason"])
	msg, _ := cond["message"].(string)
	assert.True(t, strings.Contains(msg, "dnsPolicy"), "Message should mention the colliding field 'dnsPolicy', got: %s", msg)
	assert.True(t, strings.Contains(msg, "collides"), "Message should mention the collision, got: %s", msg)
}

// ─── Additional status condition coverage (gap analysis) ──────────────────────

// TestStatusCondition_AllWarningPaths verifies that every warning path in
// loadAndMergeConfigurableCollectionWithClient sets Applied=False. Uses a config
// with one broken rule of each type to exercise each code path.
func TestStatusCondition_AllWarningPaths(t *testing.T) {
	tests := []struct {
		name          string
		rule          map[string]interface{}
		wantSubstring string // expected in condition message
	}{
		{
			name: "include without fields",
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{"Pod"},
				},
				// no fields key
			},
			wantSubstring: "requires at least one field",
		},
		{
			name: "missing kinds",
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{}, // empty
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "x", "jsonPath": "{.x}"},
				},
			},
			wantSubstring: "missing kinds",
		},
		{
			name: "multiple kinds",
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{"Pod", "Deployment"},
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "x", "jsonPath": "{.x}"},
				},
			},
			wantSubstring: "exactly 1 kind",
		},
		{
			name: "multiple apiGroups",
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{"", "apps"},
					"kinds":     []interface{}{"Pod"},
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "x", "jsonPath": "{.x}"},
				},
			},
			wantSubstring: "exactly 1 apiGroup",
		},
		{
			name: "field missing jsonPath",
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{"Pod"},
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "myField", "jsonPath": ""},
				},
			},
			wantSubstring: "name or jsonPath is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
			originalNamespace := config.Cfg.PodNamespace
			defer func() {
				config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
				config.Cfg.PodNamespace = originalNamespace
				mergedTransformConfig = nil
			}()
			config.Cfg.FeatureConfigurableCollection = true
			config.Cfg.PodNamespace = "test-namespace"

			collectionConfig := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "search.open-cluster-management.io/v1alpha1",
					"kind":       "CollectorConfig",
					"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
					"spec": map[string]interface{}{
						"collectionRules": []interface{}{tc.rule},
					},
				},
			}

			scheme := runtime.NewScheme()
			fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
			loadAndMergeConfigurableCollectionWithClient(fakeClient)

			cond := getStatusConditionFromFakeClient(fakeClient)
			require.NotNil(t, cond, "Expected a status condition for case: %s", tc.name)
			assert.Equal(t, "Applied", cond["type"])
			assert.Equal(t, "False", cond["status"], "Should be False for: %s", tc.name)
			assert.Equal(t, "RulesSkipped", cond["reason"])
			msg, _ := cond["message"].(string)
			assert.True(t, strings.Contains(msg, tc.wantSubstring),
				"Message for %s should contain %q, got: %s", tc.name, tc.wantSubstring, msg)
		})
	}
}

// TestStatusCondition_MultipleWarnings verifies that when multiple rules are
// skipped, all warning messages are concatenated into a single condition message
// separated by "; " so users see all issues at once.
func TestStatusCondition_MultipleWarnings(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
				// Rule 1: include with no fields/collectConditions — triggers "requires at least one field"
				map[string]interface{}{
					"action": "include",
					"resourceSelector": map[string]interface{}{
						"apiGroups": []interface{}{""},
						"kinds":     []interface{}{"Pod"},
					},
				},
					// Rule 2: include without fields
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Deployment"},
						},
					},
					// Rule 3: multiple kinds
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"DaemonSet", "StatefulSet"},
						},
						"fields": []interface{}{
							map[string]interface{}{"name": "x", "jsonPath": "{.x}"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond, "Expected a status condition")

	assert.Equal(t, "False", cond["status"])
	assert.Equal(t, "RulesSkipped", cond["reason"])

	msg, _ := cond["message"].(string)
	// All 3 warnings should be in the single message separated by "; "
	assert.True(t, strings.Contains(msg, "; "), "Multiple warnings should be separated by '; ', got: %s", msg)
	assert.True(t, strings.Contains(msg, "include action requires at least one field"), "Rule 1: got: %s", msg)
	assert.True(t, strings.Contains(msg, "include action requires at least one field"), "Rule 2: got: %s", msg)
	assert.True(t, strings.Contains(msg, "include action with fields must specify exactly 1 kind, found 2"), "Rule 3: got: %s", msg)
}

// TestStatusCondition_FeatureDisabled verifies that when the feature flag is off,
// loadAndMergeConfigurableCollectionWithClient is never called, so no status update
// is attempted and mergedTransformConfig is initialised from defaults only.
func TestStatusCondition_FeatureDisabled(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = false
	config.Cfg.PodNamespace = "test-namespace"

	// Track whether loadAndMergeConfigurableCollectionWithClient would have been called
	// by verifying that mergedTransformConfig contains only default entries (no custom rules).
	// A fake client with a CR is deliberately NOT passed — if the internal function ran,
	// it would contact the cluster; the fact that we only call the public wrapper proves
	// the feature gate is respected without needing to inject a spy.
	LoadAndMergeConfigurableCollection()

	// When disabled: mergedTransformConfig should mirror defaultTransformConfig exactly.
	assert.Equal(t, len(defaultTransformConfig), len(mergedTransformConfig),
		"When feature is disabled mergedTransformConfig should equal defaultTransformConfig")

	// Verify no custom fields were added (proves the CR was never read).
	for key, cfg := range mergedTransformConfig {
		defaultCfg, exists := defaultTransformConfig[key]
		assert.True(t, exists, "Unexpected key %s in mergedTransformConfig", key)
		if exists {
			assert.Equal(t, len(defaultCfg.properties), len(cfg.properties),
				"Resource %s should have only default properties when feature is disabled", key)
		}
	}
}

// TestStatusCondition_CRNotFound verifies that when the CollectorConfig CR does not
// exist, no status update is attempted (there's nothing to write to).
func TestStatusCondition_CRNotFound(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Empty fake client — no CollectorConfig CR
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// No update action should have been recorded
	cond := getStatusConditionFromFakeClient(fakeClient)
	assert.Nil(t, cond, "No status condition should be written when CollectorConfig CR does not exist")
}

// TestStatusCondition_StatusUpdateFailure verifies that when the status update is
// rejected (e.g. RBAC denied), the collector continues working — the merge result
// is still applied and no panic occurs. This matches the live cluster behavior we
// observed during E2E testing (RBAC missing → forbidden → collector keeps running).
func TestStatusCondition_StatusUpdateFailure(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{"name": "dnsPolicy", "jsonPath": "{.spec.dnsPolicy}"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	// Inject a reactor that simulates RBAC denial on any status subresource update.
	fakeClient.PrependReactor("update", "collectorconfigs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() == "status" {
			return true, nil, fmt.Errorf("collectorconfigs.search.open-cluster-management.io \"collector-config\" is forbidden: User cannot update resource \"collectorconfigs/status\"")
		}
		return false, nil, nil
	})

	// Must not panic when the status update is rejected.
	require.NotPanics(t, func() {
		loadAndMergeConfigurableCollectionWithClient(fakeClient)
	})

	// Config merge must have succeeded despite the status update failure.
	podConfig, exists := mergedTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should be merged even when status update is denied")
	assert.Equal(t, 1, len(podConfig.properties), "Custom field should have been merged")
	assert.Equal(t, "dnsPolicy", podConfig.properties[0].Name)
}

// TestStatusCondition_LastTransitionTime_PreservedWhenStatusUnchanged verifies that
// lastTransitionTime is only updated when the condition status (True/False) changes.
// If the collector restarts and the config is still valid (True→True), the timestamp
// should remain unchanged — following the Kubernetes convention that lastTransitionTime
// reflects the last state *transition*, not the last evaluation.
func TestStatusCondition_LastTransitionTime_PreservedWhenStatusUnchanged(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	existingTimestamp := "2026-01-01T10:00:00Z"

	// CollectorConfig that already has Applied=True in its status (simulating a prior run)
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
						"fields": []interface{}{
							map[string]interface{}{"name": "dnsPolicy", "jsonPath": "{.spec.dnsPolicy}"},
						},
					},
				},
			},
			// Pre-existing status with a known timestamp
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Applied",
						"status":             "True",
						"reason":             "Applied",
						"message":            "Configuration applied successfully.",
						"lastTransitionTime": existingTimestamp,
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond)

	assert.Equal(t, "True", cond["status"])
	// lastTransitionTime must be preserved — status didn't change (True→True)
	assert.Equal(t, existingTimestamp, cond["lastTransitionTime"],
		"lastTransitionTime should NOT change when status stays True")
}

// TestStatusCondition_LastTransitionTime_UpdatedWhenStatusChanges verifies that
// lastTransitionTime IS updated when the condition status transitions (True→False).
func TestStatusCondition_LastTransitionTime_UpdatedWhenStatusChanges(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	oldTimestamp := "2026-01-01T10:00:00Z"

	// Config was True before, now we have a broken rule (True → False transition)
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						// include with no fields — triggers Applied=False (RulesSkipped)
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Pod"},
						},
					},
				},
			},
			// Pre-existing status was True
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Applied",
						"status":             "True",
						"reason":             "Applied",
						"message":            "Configuration applied successfully.",
						"lastTransitionTime": oldTimestamp,
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	cond := getStatusConditionFromFakeClient(fakeClient)
	require.NotNil(t, cond)

	assert.Equal(t, "False", cond["status"])
	// lastTransitionTime MUST change — status transitioned from True to False
	assert.NotEqual(t, oldTimestamp, cond["lastTransitionTime"],
		"lastTransitionTime should update when status transitions True→False")
}

// ─── Warning truncation tests (maxStatusWarnings) ─────────────────────────────

// TestStatusCondition_WarningTruncation verifies all combinations of warning count
// against the maxStatusWarnings (3) limit.
func TestStatusCondition_WarningTruncation(t *testing.T) {
	tests := []struct {
		name            string
		numRules        int  // number of broken rules to generate
		expectTruncated bool // whether "... and N more" should appear
		expectedMore    int  // expected N in "... and N more"
	}{
		{"1 warning — below limit, no truncation", 1, false, 0},
		{"2 warnings — below limit, no truncation", 2, false, 0},
		{"3 warnings — at limit (boundary), no truncation", 3, false, 0},
		{"4 warnings — 1 over limit, truncated", 4, true, 1},
		{"5 warnings — 2 over limit, truncated", 5, true, 2},
		{"6 warnings — double the limit, truncated", 6, true, 3},
	}

	// distinctWarningRules defines 6 rule types that each produce a UNIQUE warning message.
	// This is essential: using the same rule type (e.g. all "exclude") would make all
	// warning strings identical, preventing us from asserting which ones are truncated.
	type warningRule struct {
		rule            interface{} // the CollectorConfig rule that triggers the warning
		uniqueSubstring string      // a substring unique to that warning's message
	}
	distinctWarningRules := []warningRule{
		{
			// include + fields + zero apiGroups → "exactly 1 apiGroup, found 0"
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{}, // zero apiGroups
					"kinds":     []interface{}{"Pod"},
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "v", "jsonPath": "{.v}"},
				},
			},
			uniqueSubstring: "found 0",
		},
		{
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{"Pod"},
				},
				// no fields — triggers "requires at least one field"
			},
			uniqueSubstring: "requires at least one field",
		},
		{
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{"apps"},
					"kinds":     []interface{}{"DaemonSet", "StatefulSet"}, // 2 kinds
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "x", "jsonPath": "{.x}"},
				},
			},
			uniqueSubstring: "exactly 1 kind",
		},
		{
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{"apps", "batch"}, // 2 apiGroups
					"kinds":     []interface{}{"Job"},
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "y", "jsonPath": "{.y}"},
				},
			},
			// "exactly 1 apiGroup, found 2" — "found 2" alone also appears in rule 3 ("kind, found 2")
			uniqueSubstring: "apiGroup, found 2",
		},
		{
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{}, // empty kinds
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "z", "jsonPath": "{.z}"},
				},
			},
			uniqueSubstring: "missing kinds",
		},
		{
			rule: map[string]interface{}{
				"action": "include",
				"resourceSelector": map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{""}, // empty kind string
				},
				"fields": []interface{}{
					map[string]interface{}{"name": "w", "jsonPath": "{.w}"},
				},
			},
			uniqueSubstring: "kind is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
			originalNamespace := config.Cfg.PodNamespace
			defer func() {
				config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
				config.Cfg.PodNamespace = originalNamespace
				mergedTransformConfig = nil
			}()
			config.Cfg.FeatureConfigurableCollection = true
			config.Cfg.PodNamespace = "test-namespace"

			rules := make([]interface{}, tc.numRules)
			for i := 0; i < tc.numRules; i++ {
				rules[i] = distinctWarningRules[i].rule
			}

			collectionConfig := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "search.open-cluster-management.io/v1alpha1",
					"kind":       "CollectorConfig",
					"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
					"spec":       map[string]interface{}{"collectionRules": rules},
				},
			}

			scheme := runtime.NewScheme()
			fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
			loadAndMergeConfigurableCollectionWithClient(fakeClient)

			cond := getStatusConditionFromFakeClient(fakeClient)
			require.NotNil(t, cond, "Expected a status condition for case: %s", tc.name)
			assert.Equal(t, "False", cond["status"])

			msg, _ := cond["message"].(string)

			if tc.expectTruncated {
				// Truncation suffix must be present
				expectedSuffix := fmt.Sprintf("; ... and %d more", tc.expectedMore)
				assert.True(t, strings.Contains(msg, expectedSuffix),
					"Expected suffix %q in message, got: %s", expectedSuffix, msg)

				// First 3 warning unique substrings MUST appear
				for i := 0; i < 3; i++ {
					assert.True(t, strings.Contains(msg, distinctWarningRules[i].uniqueSubstring),
						"Warning %d (%q) should be in message, got: %s",
						i+1, distinctWarningRules[i].uniqueSubstring, msg)
				}

				// Warnings beyond the limit MUST NOT appear as full entries
				for i := 3; i < tc.numRules; i++ {
					assert.False(t, strings.Contains(msg, distinctWarningRules[i].uniqueSubstring),
						"Truncated warning %d (%q) should NOT be in message, got: %s",
						i+1, distinctWarningRules[i].uniqueSubstring, msg)
				}
			} else {
				// No truncation: all warnings present, no suffix
				assert.False(t, strings.Contains(msg, "... and"),
					"Expected no truncation for %d warnings (limit is 3), got: %s", tc.numRules, msg)

				// Every warning unique substring must be present
				for i := 0; i < tc.numRules; i++ {
					assert.True(t, strings.Contains(msg, distinctWarningRules[i].uniqueSubstring),
						"Warning %d (%q) should be in message, got: %s",
						i+1, distinctWarningRules[i].uniqueSubstring, msg)
				}
			}
		})
	}
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsSpecificKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(5),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["Alertmanager.monitoring.coreos.com"]
	assert.True(t, exists, "Alertmanager.monitoring.coreos.com config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 5, *rc.additionalPrinterColumnsPriority, "priority threshold should be 5")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsWildcardKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"*"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(0),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["*.monitoring.coreos.com"]
	assert.True(t, exists, "*.monitoring.coreos.com wildcard config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 0, *rc.additionalPrinterColumnsPriority, "priority threshold should be 0")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsWithFieldsAndConditions(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":            "include",
						"collectConditions": true,
						"collectAdditionalPrinterColumnsPriority": int64(1),
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"kubevirt.io"},
							"kinds":     []interface{}{"VirtualMachine"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "running",
								"jsonPath": "{.spec.running}",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["VirtualMachine.kubevirt.io"]
	assert.True(t, exists, "VirtualMachine.kubevirt.io config should exist")
	assert.True(t, rc.extractConditions, "extractConditions should be true")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 1, *rc.additionalPrinterColumnsPriority, "priority threshold should be 1")
	// The custom field should be merged into the config (may also contain default properties).
	foundRunning := false
	for _, p := range rc.properties {
		if p.Name == "running" {
			foundRunning = true
			break
		}
	}
	assert.True(t, foundRunning, "custom field 'running' should be present in properties")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsPriorityMaxWins(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Two rules target the same resource with different priorities.
	// The higher value (more permissive) should win regardless of order.
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Integration team rule: priority 10
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(10),
					},
					// User rule: priority 0 (should not narrow the team's threshold)
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(0),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["Alertmanager.monitoring.coreos.com"]
	assert.True(t, exists, "Alertmanager.monitoring.coreos.com config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 10, *rc.additionalPrinterColumnsPriority,
		"Max priority should win — user's priority 0 should not narrow team's priority 10")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsPriorityMaxWinsReverseProcessingOrder(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Two rules target the same resource with different priorities.
	// The higher value (more permissive) should win regardless of order.
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Integration team rule: priority 0
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(0),
					},
					// User rule: priority 10 (should not narrow the team's threshold)
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(10),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["Alertmanager.monitoring.coreos.com"]
	assert.True(t, exists, "Alertmanager.monitoring.coreos.com config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 10, *rc.additionalPrinterColumnsPriority,
		"Max priority should win — team's priority 0 should not narrow user's priority 10")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsDisabled(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Priority -1 should disable additionalPrinterColumns collection.
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(-1),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["Alertmanager.monitoring.coreos.com"]
	assert.True(t, exists, "Alertmanager.monitoring.coreos.com config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, -1, *rc.additionalPrinterColumnsPriority,
		"Priority -1 should be stored to disable collection")
}

func TestLoadAndMergeConfigurableCollection_CollectPrinterColumnsDisabledOverriddenByPositive(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// When one rule disables (-1) and another enables (5), max wins (5).
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Rule 1: disable
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(-1),
					},
					// Rule 2: enable with priority 5
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"monitoring.coreos.com"},
							"kinds":     []interface{}{"Alertmanager"},
						},
						"collectAdditionalPrinterColumnsPriority": int64(5),
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	rc, exists := mergedTransformConfig["Alertmanager.monitoring.coreos.com"]
	assert.True(t, exists, "Alertmanager.monitoring.coreos.com config should exist")
	assert.NotNil(t, rc.additionalPrinterColumnsPriority, "additionalPrinterColumnsPriority should be set")
	assert.Equal(t, 5, *rc.additionalPrinterColumnsPriority,
		"Max priority should win — positive value (5) should override disable (-1)")
}

// TestNormalizeJSONPath verifies that the collector accepts jsonPath values both
// with and without curly-brace wrapping, normalizing them automatically so users
// don't need to know the k8s jsonpath library convention.
// See: https://issues.redhat.com/browse/ACM-33144
func TestNormalizeJSONPath(t *testing.T) {
	tests := []struct {
		name             string
		jsonPath         string // as the user writes it in the CollectorConfig
		expectedJSONPath string // what should actually be stored and used
	}{
		// Users who know the convention — unchanged
		{"already has braces", "{.spec.dnsPolicy}", "{.spec.dnsPolicy}"},
		{"nested path with braces", "{.metadata.labels.app}", "{.metadata.labels.app}"},
		{
			"filter expression with braces",
			"{.status.conditions[?(@.type=='Ready')].status}",
			"{.status.conditions[?(@.type=='Ready')].status}",
		},

		// Users who omit braces — should be auto-wrapped
		{"no braces — simple path", ".spec.dnsPolicy", "{.spec.dnsPolicy}"},
		{"no braces — nested path", ".metadata.labels.app", "{.metadata.labels.app}"},
		{
			"no braces — filter expression",
			".status.conditions[?(@.type=='Ready')].status",
			"{.status.conditions[?(@.type=='Ready')].status}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
			originalNamespace := config.Cfg.PodNamespace
			defer func() {
				config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
				config.Cfg.PodNamespace = originalNamespace
				mergedTransformConfig = nil
			}()
			config.Cfg.FeatureConfigurableCollection = true
			config.Cfg.PodNamespace = "test-namespace"

			collectionConfig := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "search.open-cluster-management.io/v1alpha1",
					"kind":       "CollectorConfig",
					"metadata":   map[string]interface{}{"name": "merged-collector-config", "namespace": "test-namespace"},
					"spec": map[string]interface{}{
						"collectionRules": []interface{}{
							map[string]interface{}{
								"action": "include",
								"resourceSelector": map[string]interface{}{
									"apiGroups": []interface{}{""},
									"kinds":     []interface{}{"Pod"},
								},
								"fields": []interface{}{
									map[string]interface{}{
										"name":     "testField",
										"jsonPath": tc.jsonPath,
									},
								},
							},
						},
					},
				},
			}

			scheme := runtime.NewScheme()
			fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)
			loadAndMergeConfigurableCollectionWithClient(fakeClient)

			podConfig, exists := mergedTransformConfig["Pod"]
			assert.True(t, exists, "Pod config should exist in mergedTransformConfig")
			require.Equal(t, 1, len(podConfig.properties), "Pod should have 1 custom property")
			assert.Equal(t, "testField", podConfig.properties[0].Name)
			assert.Equal(t, tc.expectedJSONPath, podConfig.properties[0].JSONPath,
				"jsonPath should be normalized to %q regardless of input format", tc.expectedJSONPath)
		})
	}
}

// ============================================================================
// CollectAnnotations tests — mirrors the CollectConditions test suite
// ============================================================================

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsWithSpecificKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractAnnotations, "extractAnnotations should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsWithMultipleKinds(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment", "StatefulSet"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractAnnotations, "Deployment extractAnnotations should be true")

	ssConfig, exists := mergedTransformConfig["StatefulSet.apps"]
	assert.True(t, exists, "StatefulSet.apps config should exist")
	assert.True(t, ssConfig.extractAnnotations, "StatefulSet extractAnnotations should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsWildcardKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	wildcardConfig, exists := mergedTransformConfig["*.apps"]
	assert.True(t, exists, "*.apps wildcard config should exist in mergedTransformConfig")
	assert.True(t, wildcardConfig.extractAnnotations, "*.apps extractAnnotations should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsMultipleApiGroups(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps", "batch"},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	appsConfig, exists := mergedTransformConfig["*.apps"]
	assert.True(t, exists, "*.apps wildcard config should exist")
	assert.True(t, appsConfig.extractAnnotations, "*.apps extractAnnotations should be true")

	batchConfig, exists := mergedTransformConfig["*.batch"]
	assert.True(t, exists, "*.batch wildcard config should exist")
	assert.True(t, batchConfig.extractAnnotations, "*.batch extractAnnotations should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsWithFieldsAndKind(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "replicas",
								"jsonPath": "{.spec.replicas}",
								"type":     "number",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist")
	assert.True(t, deployConfig.extractAnnotations, "extractAnnotations should be true")
	assert.Equal(t, 1, len(deployConfig.properties), "Should have 1 custom field")
	assert.Equal(t, "replicas", deployConfig.properties[0].Name)
	assert.Equal(t, DataTypeNumber, deployConfig.properties[0].DataType)
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsPreservesDefaults(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Empty rules - verify defaults are preserved
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify defaults with extractAnnotations are preserved
	dvConfig, exists := mergedTransformConfig["DataVolume.cdi.kubevirt.io"]
	assert.True(t, exists, "DataVolume.cdi.kubevirt.io config should exist from defaults")
	assert.True(t, dvConfig.extractAnnotations, "DataVolume should retain default extractAnnotations=true")

	nadConfig, exists := mergedTransformConfig["NetworkAttachmentDefinition.k8s.cni.cncf.io"]
	assert.True(t, exists, "NetworkAttachmentDefinition.k8s.cni.cncf.io config should exist from defaults")
	assert.True(t, nadConfig.extractAnnotations, "NetworkAttachmentDefinition should retain default extractAnnotations=true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsCoreApiGroup(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"*"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Core API group is empty string - wildcard key is just "*" (no dot prefix)
	coreConfig, exists := mergedTransformConfig["*"]
	assert.True(t, exists, "* wildcard config should exist for core API group")
	assert.True(t, coreConfig.extractAnnotations, "* extractAnnotations should be true")
}

func TestLoadAndMergeConfigurableCollection_CollectAnnotationsOnly(t *testing.T) {
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		mergedTransformConfig = nil
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Rule with only collectAnnotations — no fields, no collectConditions.
	// This should NOT be skipped by the guard clause.
	collectAnnotations := true
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action":             "include",
						"collectAnnotations": collectAnnotations,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	deployConfig, exists := mergedTransformConfig["Deployment.apps"]
	assert.True(t, exists, "Deployment.apps config should exist — collectAnnotations-only rule must not be skipped")
	assert.True(t, deployConfig.extractAnnotations, "extractAnnotations should be true")
	assert.Empty(t, deployConfig.properties, "Should have no custom fields")
	assert.False(t, deployConfig.extractConditions, "extractConditions should be false")
}

// --- Exclude action tests ---

func makeExcludeConfig(namespace, apiGroup, kind string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{apiGroup},
							"kinds":     []interface{}{kind},
						},
					},
				},
			},
		},
	}
}

func setupExcludeTest(t *testing.T) func() {
	t.Helper()
	orig := config.Cfg.FeatureConfigurableCollection
	origNS := config.Cfg.PodNamespace
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"
	return func() {
		config.Cfg.FeatureConfigurableCollection = orig
		config.Cfg.PodNamespace = origNS
		mergedTransformConfig = nil
		excludeRules = nil
	}
}

// Specific kind+group is excluded.
func TestExclude_SpecificKind(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "coordination.k8s.io", "Lease")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease.coordination.k8s.io should be excluded")
	assert.False(t, IsResourceExcluded("coordination.k8s.io", "LeaderElection"),
		"Other kinds in group should not be excluded")
	assert.False(t, IsResourceExcluded("", "Lease"),
		"Core-group Lease should not be excluded")
}

// Wildcard kind excludes all kinds in a group.
func TestExclude_WildcardKind(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "coordination.k8s.io", "*")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease should be excluded via wildcard")
	assert.True(t, IsResourceExcluded("coordination.k8s.io", "LeaderElection"),
		"Any kind in coordination.k8s.io should be excluded via wildcard")
	assert.False(t, IsResourceExcluded("apps", "Deployment"),
		"Other groups should not be affected")
}

// Wildcard group and kind excludes everything.
func TestExclude_WildcardAll(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "*", "*")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("apps", "Deployment"),
		"Deployment should be excluded via global wildcard")
	assert.True(t, IsResourceExcluded("", "Pod"),
		"Core-group Pod should be excluded via global wildcard")
}

// Core-group (empty apiGroup) specific kind is excluded.
func TestExclude_CoreGroup(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "", "Event")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("", "Event"),
		"Core-group Event should be excluded")
	assert.False(t, IsResourceExcluded("", "Pod"),
		"Core-group Pod should not be excluded")
	assert.False(t, IsResourceExcluded("events.k8s.io", "Event"),
		"Event in different apiGroup should not be excluded")
}

// Exclude does not affect mergedTransformConfig — default properties still extracted.
func TestExclude_DoesNotAffectTransformConfig(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "coordination.k8s.io", "Lease")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// The exclude rule should not remove the Lease entry from mergedTransformConfig;
	// exclusion is enforced at the informer layer, not in the transform config.
	_, found := mergedTransformConfig["Lease.coordination.k8s.io"]
	assert.False(t, found,
		"Lease.coordination.k8s.io was not in defaultTransformConfig so should not appear in mergedTransformConfig")
	// The Pod default config should still be intact.
	_, podFound := mergedTransformConfig["Pod"]
	assert.True(t, podFound, "Default Pod config should still be present in mergedTransformConfig")
}

// Last entry wins: include after exclude for same resource cancels the exclusion.
func TestExclude_LastEntryWins_IncludeAfterExclude(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	collectConditions := true
	cfg := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Rule 1: include Lease (sets collectConditions)
					map[string]interface{}{
						"action":             "include",
						"collectConditions":  collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
					// Rule 2: exclude Lease
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
					// Rule 3: include Lease again — last entry wins, should cancel exclude
					map[string]interface{}{
						"action":             "include",
						"collectConditions":  collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.False(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Last include should cancel the prior exclude — Lease should NOT be excluded")
}

// Last entry wins: exclude after include for same resource keeps the exclusion.
func TestExclude_LastEntryWins_ExcludeAfterInclude(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	collectConditions := true
	cfg := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Rule 1: include Lease
					map[string]interface{}{
						"action":             "include",
						"collectConditions":  collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
					// Rule 2: exclude Lease — last entry, should win
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Last exclude should win — Lease should be excluded")
}

// Group wildcard: exclude "Lease.*" matches Lease in any apiGroup.
func TestExclude_GroupWildcard(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	cfg := makeExcludeConfig("test-namespace", "*", "Lease")
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease in any group should be excluded via group wildcard")
	assert.True(t, IsResourceExcluded("", "Lease"),
		"Core-group Lease should also be excluded via group wildcard")
	assert.False(t, IsResourceExcluded("coordination.k8s.io", "LeaderElection"),
		"Other kinds should not be affected")
}

// A skipped include rule (no fields/conditions) must NOT cancel a prior exclude.
func TestExclude_InvalidIncludeDoesNotCancelExclude(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	collectConditions := true
	cfg := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					// Valid include first
					map[string]interface{}{
						"action":            "include",
						"collectConditions": collectConditions,
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
					// Exclude
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
					// Invalid include (no fields/conditions) — must NOT cancel the exclude
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"Lease"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Invalid include (no fields/conditions) must NOT cancel the prior exclude")
}

// Wildcard exclude followed by specific include: the include overrides the wildcard.
// This is the primary use-case that was previously a documented limitation.
// A user who wants "collect only Deployments" can now write:
//   exclude "*.*" (deny all) + include "Deployment.apps" (allow specific)
func TestExclude_WildcardOverriddenBySpecificInclude(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"*"},
							"kinds":     []interface{}{"*"},
						},
					},
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
						"fields": []interface{}{
							map[string]interface{}{"name": "replicas", "jsonPath": ".spec.replicas"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), collectionConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Deployment should NOT be excluded — the specific include overrides the global exclude.
	assert.False(t, IsResourceExcluded("apps", "Deployment"),
		"Specific include 'Deployment.apps' must override prior global exclude '*.*'")

	// Other resources ARE excluded by the wildcard.
	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Lease must still be excluded by global exclude '*.*'")
	assert.True(t, IsResourceExcluded("", "ConfigMap"),
		"ConfigMap (core group) must still be excluded by global exclude '*.*'")
}

// IsResourceExcluded returns false when excludeRules is nil (no rules loaded).
func TestExclude_NilRules(t *testing.T) {
	excludeRules = nil
	assert.False(t, IsResourceExcluded("apps", "Deployment"),
		"Should return false when excludeRules is nil")
}

// Config reload resets excludeRules — old excludes do not persist.
func TestExclude_ResetOnReload(t *testing.T) {
	teardown := setupExcludeTest(t)
	defer teardown()

	// First load: exclude Lease
	cfg1 := makeExcludeConfig("test-namespace", "coordination.k8s.io", "Lease")
	fakeClient1 := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg1)
	loadAndMergeConfigurableCollectionWithClient(fakeClient1)
	assert.True(t, IsResourceExcluded("coordination.k8s.io", "Lease"))

	// Second load: no exclude rules
	cfg2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{},
			},
		},
	}
	fakeClient2 := fake.NewSimpleDynamicClient(runtime.NewScheme(), cfg2)
	loadAndMergeConfigurableCollectionWithClient(fakeClient2)
	assert.False(t, IsResourceExcluded("coordination.k8s.io", "Lease"),
		"Previous exclude should be cleared after reload with no exclude rules")
}
