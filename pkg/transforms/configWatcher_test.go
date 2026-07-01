package transforms

import (
	"sort"
	"sync"
	"testing"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestDiffConfigs(t *testing.T) {
	baseCfg := ResourceConfig{
		properties: []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}},
	}
	modifiedCfg := ResourceConfig{
		properties: []ExtractProperty{
			{Name: "name", JSONPath: "{.metadata.name}"},
			{Name: "status", JSONPath: "{.status.phase}"},
		},
	}
	conditionsOnCfg := ResourceConfig{
		properties:        []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}},
		extractConditions: true,
	}
	priority0 := 0
	priority5 := 5
	printerCfg0 := ResourceConfig{
		properties:                       []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}},
		additionalPrinterColumnsPriority: &priority0,
	}
	printerCfg5 := ResourceConfig{
		properties:                       []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}},
		additionalPrinterColumnsPriority: &priority5,
	}

	tests := []struct {
		name     string
		old      map[string]ResourceConfig
		new      map[string]ResourceConfig
		expected []string
	}{
		{
			name:     "identical configs",
			old:      map[string]ResourceConfig{"Pod": baseCfg},
			new:      map[string]ResourceConfig{"Pod": baseCfg},
			expected: []string{},
		},
		{
			name:     "key added",
			old:      map[string]ResourceConfig{},
			new:      map[string]ResourceConfig{"Pod": baseCfg},
			expected: []string{"Pod"},
		},
		{
			name:     "key removed",
			old:      map[string]ResourceConfig{"Pod": baseCfg},
			new:      map[string]ResourceConfig{},
			expected: []string{"Pod"},
		},
		{
			name:     "key changed - property added",
			old:      map[string]ResourceConfig{"Pod": baseCfg},
			new:      map[string]ResourceConfig{"Pod": modifiedCfg},
			expected: []string{"Pod"},
		},
		{
			name:     "key changed - extractConditions toggled",
			old:      map[string]ResourceConfig{"Pod": baseCfg},
			new:      map[string]ResourceConfig{"Pod": conditionsOnCfg},
			expected: []string{"Pod"},
		},
		{
			name:     "key changed - printerColumnsPriority changed",
			old:      map[string]ResourceConfig{"Pod": printerCfg0},
			new:      map[string]ResourceConfig{"Pod": printerCfg5},
			expected: []string{"Pod"},
		},
		{
			name:     "multiple changes",
			old:      map[string]ResourceConfig{"Pod": baseCfg, "Secret": baseCfg},
			new:      map[string]ResourceConfig{"Pod": baseCfg, "Secret": modifiedCfg, "Node": baseCfg},
			expected: []string{"Node", "Secret"},
		},
		{
			name:     "both nil",
			old:      nil,
			new:      nil,
			expected: []string{},
		},
		{
			name:     "old nil new populated",
			old:      nil,
			new:      map[string]ResourceConfig{"Pod": baseCfg},
			expected: []string{"Pod"},
		},
		{
			name:     "wildcard key changed",
			old:      map[string]ResourceConfig{"*.apps": baseCfg},
			new:      map[string]ResourceConfig{"*.apps": conditionsOnCfg},
			expected: []string{"*.apps"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := diffConfigs(tc.old, tc.new)
			sort.Strings(result)
			sort.Strings(tc.expected)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSnapshotConfig(t *testing.T) {
	// Save and restore global state.
	origConfig := mergedTransformConfig
	defer func() {
		configMu.Lock()
		mergedTransformConfig = origConfig
		configMu.Unlock()
	}()

	original := map[string]ResourceConfig{
		"Pod": {
			properties:        []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}},
			extractConditions: true,
		},
	}
	configMu.Lock()
	mergedTransformConfig = original
	configMu.Unlock()

	snapshot := snapshotConfig()

	// Verify it matches the original.
	assert.Equal(t, len(original), len(snapshot))
	assert.Equal(t, original["Pod"].extractConditions, snapshot["Pod"].extractConditions)

	// Mutate the snapshot — should NOT affect the global.
	snapshot["Pod"] = ResourceConfig{extractConditions: false}
	snapshot["NewKey"] = ResourceConfig{}

	configMu.RLock()
	assert.True(t, mergedTransformConfig["Pod"].extractConditions, "mutating snapshot should not affect global")
	_, exists := mergedTransformConfig["NewKey"]
	assert.False(t, exists, "adding key to snapshot should not affect global")
	configMu.RUnlock()
}

func TestSnapshotConfig_Concurrent(t *testing.T) {
	// Save and restore global state.
	origConfig := mergedTransformConfig
	origFeature := config.Cfg.FeatureConfigurableCollection
	origNamespace := config.Cfg.PodNamespace
	defer func() {
		config.Cfg.FeatureConfigurableCollection = origFeature
		config.Cfg.PodNamespace = origNamespace
		configMu.Lock()
		mergedTransformConfig = origConfig
		configMu.Unlock()
	}()

	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-ns"

	// Initialize with a known config.
	configMu.Lock()
	mergedTransformConfig = map[string]ResourceConfig{
		"Pod": {properties: []ExtractProperty{{Name: "name", JSONPath: "{.metadata.name}"}}},
	}
	configMu.Unlock()

	// Create a fake client that returns no CollectorConfig (forces default config).
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var wg sync.WaitGroup
	const numReaders = 10

	// Launch readers concurrently with a writer.
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap := snapshotConfig()
			// Just verify no panic and we get a non-nil map.
			if snap == nil {
				t.Error("snapshotConfig returned nil")
			}
		}()
	}

	// Writer goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		loadAndMergeConfigurableCollectionWithClient(fakeClient)
	}()

	wg.Wait()
}

// Helper to save and restore global state used by reloadAndResync tests.
func saveAndRestoreConfigState(t *testing.T) {
	t.Helper()
	origConfig := mergedTransformConfig
	origFeature := config.Cfg.FeatureConfigurableCollection
	origNamespace := config.Cfg.PodNamespace
	t.Cleanup(func() {
		config.Cfg.FeatureConfigurableCollection = origFeature
		config.Cfg.PodNamespace = origNamespace
		configMu.Lock()
		mergedTransformConfig = origConfig
		configMu.Unlock()
	})
	config.Cfg.FeatureConfigurableCollection = true
	config.Cfg.PodNamespace = "test-ns"
}

func TestReloadAndResync_NoChange(t *testing.T) {
	saveAndRestoreConfigState(t)

	// Set initial config and create a fake client with no CollectorConfig CR.
	// Reloading without a CR produces defaults, so pre-set to defaults.
	configMu.Lock()
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
	configMu.Unlock()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackKeys []string
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackKeys = keys
	})

	cw.reloadAndResync()

	assert.Nil(t, callbackKeys, "callback should not be called when config is unchanged")
}

func TestReloadAndResync_ConfigAdded(t *testing.T) {
	saveAndRestoreConfigState(t)

	// Start with defaults — no custom fields for Pod.
	configMu.Lock()
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
	configMu.Unlock()

	// Create a CollectorConfig that adds a custom field to Pod.
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-ns",
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

	var callbackKeys []string
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackKeys = keys
	})

	cw.reloadAndResync()

	// Pod config added, therefore we resync Pod resources to keep data fresh
	assert.NotNil(t, callbackKeys, "callback should be called when config changes")
	assert.Contains(t, callbackKeys, "Pod", "Pod should be in the affected keys")
}

func TestReloadAndResync_ConfigRemoved(t *testing.T) {
	saveAndRestoreConfigState(t)

	// Start with a config that has a custom field for Pod.
	initialConfig := deepCopyTransformConfig(defaultTransformConfig)
	podCfg := initialConfig["Pod"]
	podCfg.properties = append(podCfg.properties, ExtractProperty{Name: "custom", JSONPath: "{.spec.custom}"})
	initialConfig["Pod"] = podCfg

	configMu.Lock()
	mergedTransformConfig = initialConfig
	configMu.Unlock()

	// Fake client with no CollectorConfig — reload reverts to defaults.
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackKeys []string
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackKeys = keys
	})

	cw.reloadAndResync()

	// Pod config removed, therefore we resync Pod resources to keep data fresh
	assert.NotNil(t, callbackKeys, "callback should be called when config changes")
	assert.Contains(t, callbackKeys, "Pod", "Pod should be in the affected keys")
}

func TestReloadAndResync_MultipleResourcesChanged(t *testing.T) {
	saveAndRestoreConfigState(t)

	configMu.Lock()
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
	configMu.Unlock()

	// CollectorConfig that adds fields to both Pod and a custom resource.
	collectionConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "merged-collector-config",
				"namespace": "test-ns",
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
					map[string]interface{}{
						"action": "include",
						"resourceSelector": map[string]interface{}{
							"apiGroups": []interface{}{"apps"},
							"kinds":     []interface{}{"Deployment"},
						},
						"collectConditions": true,
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, collectionConfig)

	var callbackKeys []string
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackKeys = keys
	})

	cw.reloadAndResync()

	// Multiple resources changed in one config, therefore we resync both resources to keep data fresh
	assert.NotNil(t, callbackKeys)
	sort.Strings(callbackKeys)
	assert.Contains(t, callbackKeys, "Pod")
	assert.Contains(t, callbackKeys, "Deployment.apps")
}
