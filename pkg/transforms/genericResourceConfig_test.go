// Copyright Contributors to the Open Cluster Management project

package transforms

import (
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
	originalPodConfig := defaultTransformConfig["Pod"]
	originalSearchConfig := defaultTransformConfig["Search.search.open-cluster-management.io"]
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		config.Cfg.PodNamespace = originalNamespace
		// Restore defaultTransformConfig to original state
		defaultTransformConfig["Pod"] = originalPodConfig
		defaultTransformConfig["Search.search.open-cluster-management.io"] = originalSearchConfig
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-namespace"

	// Create a mock CollectionConfig
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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
								"type":     "DataTypeString",
							},
						},
					},
					map[string]interface{}{
						"action": "Include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"search.open-cluster-management.io"},
							"kinds":     []interface{}{"Search"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "searchPGStorage",
								"jsonPath": "{.spec.dbStorage.size}",
								"type":     "DataTypeBytes",
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

	// Verify Pod config was updated
	podConfig, exists := defaultTransformConfig["Pod"]
	assert.True(t, exists, "Pod config should exist")
	assert.Equal(t, 2, len(podConfig.properties), "Pod should have 2 custom properties")
	assert.Equal(t, "user_dnsPolicy", podConfig.properties[0].Name)
	assert.Equal(t, "{.spec.dnsPolicy}", podConfig.properties[0].JSONPath)
	assert.Equal(t, "user_enableServiceLinks", podConfig.properties[1].Name)
	assert.Equal(t, DataTypeString, podConfig.properties[1].DataType)

	// Verify Search config was updated
	searchConfig, exists := defaultTransformConfig["Search.search.open-cluster-management.io"]
	assert.True(t, exists, "Search config should exist")
	assert.Equal(t, 1, len(searchConfig.properties), "Search should have 1 custom property")
	assert.Equal(t, "user_searchPGStorage", searchConfig.properties[0].Name)
	assert.Equal(t, DataTypeBytes, searchConfig.properties[0].DataType)
}

// FUTURE: this should eventually appropriately include when search-collector-config merged
func TestLoadAndMergeConfigurableCollection_SkipExcludeActions(t *testing.T) {
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Exclude",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"coordination.k8s.io"},
							"kinds":     []interface{}{"leases"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	// Store original length
	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Exclude actions should not modify config")
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Include actions without fields should not modify config")
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added (rule should be skipped)
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Rules with multiple kinds should be skipped")
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify no new entries were added (rule should be skipped)
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Rules with multiple apiGroups should be skipped")
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

	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify nothing changed
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Config should not change when feature is disabled")
}

func TestLoadAndMergeConfigurableCollection_PublicMethodRespectsFeatureFlag(t *testing.T) {
	// This test verifies the public method actually checks the feature flag
	originalFeatureFlag := config.Cfg.FeatureConfigurableCollection
	originalPodConfig := defaultTransformConfig["Pod"]
	defer func() {
		config.Cfg.FeatureConfigurableCollection = originalFeatureFlag
		defaultTransformConfig["Pod"] = originalPodConfig
	}()

	// Test with feature DISABLED
	config.Cfg.FeatureConfigurableCollection = false
	originalLen := len(defaultTransformConfig)

	// Call the PUBLIC method (not the internal helper)
	LoadAndMergeConfigurableCollection()

	// Should not have attempted to load anything
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Public method should respect feature flag when disabled")

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

	originalLen := len(defaultTransformConfig)

	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Verify nothing changed (should log warning but not fail)
	assert.Equal(t, originalLen, len(defaultTransformConfig), "Config should not change when resource not found")
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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
	secretConfig, exists := defaultTransformConfig["Secret"]
	assert.True(t, exists, "Secret config should exist")
	assert.Equal(t, 1, len(secretConfig.properties), "Secret should have 1 custom property")
	assert.Equal(t, "user_data", secretConfig.properties[0].Name)
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"test.io"},
							"kinds":     []interface{}{"TestResource"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name":     "stringField",
								"jsonPath": "{.spec.stringField}",
								"type":     "DataTypeString",
							},
							map[string]interface{}{
								"name":     "numberField",
								"jsonPath": "{.spec.numberField}",
								"type":     "DataTypeNumber",
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
	testResourceConfig, exists := defaultTransformConfig["TestResource.test.io"]
	assert.True(t, exists, "TestResource config should exist")
	assert.Equal(t, 2, len(testResourceConfig.properties), "TestResource should have 2 custom properties")

	// Verify DataTypeString
	assert.Equal(t, "user_stringField", testResourceConfig.properties[0].Name)
	assert.Equal(t, DataTypeString, testResourceConfig.properties[0].DataType)

	// Verify DataTypeNumber
	assert.Equal(t, "user_numberField", testResourceConfig.properties[1].Name)
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

	// Create a CollectionConfig without spec
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			// No spec field
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should not modify config when spec is missing
	assert.Equal(t, originalLen, len(defaultTransformConfig))
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

	// Create a CollectionConfig with invalid spec type
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": "not-a-map", // Invalid type
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should not modify config when spec is not a map
	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": "not-an-array", // Invalid type
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
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

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
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

	originalLen := len(defaultTransformConfig)
	loadAndMergeConfigurableCollectionWithClient(fakeClient)

	// Should skip rules with empty kind string
	assert.Equal(t, originalLen, len(defaultTransformConfig))
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
			"kind":       "CollectionConfig",
			"metadata": map[string]interface{}{
				"name":      "collection-config",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"collectionRules": []interface{}{
					map[string]interface{}{
						"action": "Include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{""},
							"kinds":     []interface{}{"Service"},
						},
						"fields": []interface{}{
							map[string]interface{}{
								"name": "validField",
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
	serviceConfig, exists := defaultTransformConfig["Service"]
	assert.True(t, exists)
	assert.Equal(t, 1, len(serviceConfig.properties), "Should only have 1 valid field")
	assert.Equal(t, "user_validField", serviceConfig.properties[0].Name)
}

func TestStringToDataType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DataType
	}{
		{"DataTypeBytes", "DataTypeBytes", DataTypeBytes},
		{"DataTypeSlice", "DataTypeSlice", DataTypeSlice},
		{"DataTypeString", "DataTypeString", DataTypeString},
		{"DataTypeNumber", "DataTypeNumber", DataTypeNumber},
		{"DataTypeMapString", "DataTypeMapString", DataTypeMapString},
		{"Empty String", "", DataTypeString},             // Default
		{"Unknown Value", "UnknownType", DataTypeString}, // Default
		{"Invalid Case", "datatypestring", DataTypeString}, // Case sensitive, should default
		{"Random String", "foobar", DataTypeString},      // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToDataType(tt.input)
			assert.Equal(t, tt.expected, result, "DataType mismatch for input: %s", tt.input)
		})
	}
}
