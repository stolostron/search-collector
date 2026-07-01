package transforms

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
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

// newFakeConfigObj creates an unstructured CollectorConfig with the given generation.
func newFakeConfigObj(generation int64) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":       "merged-collector-config",
				"namespace":  "test-ns",
				"generation": generation,
			},
			"spec": map[string]interface{}{},
		},
	}
}

func TestProcessEvents_Modified(t *testing.T) {
	saveAndRestoreConfigState(t)
	// Start with a config that differs from defaults so the reload produces a diff.
	modifiedConfig := deepCopyTransformConfig(defaultTransformConfig)
	podCfg := modifiedConfig["Pod"]
	podCfg.properties = append(podCfg.properties, ExtractProperty{Name: "extra", JSONPath: "{.spec.extra}"})
	modifiedConfig["Pod"] = podCfg
	configMu.Lock()
	mergedTransformConfig = modifiedConfig
	configMu.Unlock()

	// Fake client with no CR — reload reverts to defaults, producing a diff on "Pod".
	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackCount int
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackCount++
	})

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	// Send a Modified event with generation 1.
	fakeWatcher.Modify(newFakeConfigObj(1))

	// Close watcher to end processEvents.
	fakeWatcher.Stop()
	result := <-done

	assert.True(t, result, "processEvents should return true when watch ends")
	assert.Equal(t, int64(1), cw.lastSeenGeneration)
	assert.Equal(t, 1, callbackCount, "callback should be called once for Modified")
}

func TestProcessEvents_ModifiedSkipsStatusOnly(t *testing.T) {
	saveAndRestoreConfigState(t)
	// Start with config that differs from defaults so gen-2 reload produces a diff.
	modifiedConfig := deepCopyTransformConfig(defaultTransformConfig)
	podCfg := modifiedConfig["Pod"]
	podCfg.properties = append(podCfg.properties, ExtractProperty{Name: "extra", JSONPath: "{.spec.extra}"})
	modifiedConfig["Pod"] = podCfg
	configMu.Lock()
	mergedTransformConfig = modifiedConfig
	configMu.Unlock()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackCount int
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackCount++
	})
	// Pre-set generation to simulate having already seen gen 1.
	cw.lastSeenGeneration = 1

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	// Send Modified with same generation (status-only update).
	fakeWatcher.Modify(newFakeConfigObj(1))

	// Send Modified with bumped generation (spec change).
	fakeWatcher.Modify(newFakeConfigObj(2))

	fakeWatcher.Stop()
	<-done

	assert.Equal(t, int64(2), cw.lastSeenGeneration)
	assert.Equal(t, 1, callbackCount, "callback should only be called for the spec change, not the status-only update")
}

func TestProcessEvents_Added(t *testing.T) {
	saveAndRestoreConfigState(t)
	modifiedConfig := deepCopyTransformConfig(defaultTransformConfig)
	podCfg := modifiedConfig["Pod"]
	podCfg.properties = append(podCfg.properties, ExtractProperty{Name: "extra", JSONPath: "{.spec.extra}"})
	modifiedConfig["Pod"] = podCfg
	configMu.Lock()
	mergedTransformConfig = modifiedConfig
	configMu.Unlock()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackCount int
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackCount++
	})

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	fakeWatcher.Add(newFakeConfigObj(1))
	fakeWatcher.Stop()
	<-done

	assert.Equal(t, int64(1), cw.lastSeenGeneration)
	assert.Equal(t, 1, callbackCount)
}

func TestProcessEvents_Deleted(t *testing.T) {
	saveAndRestoreConfigState(t)
	modifiedConfig := deepCopyTransformConfig(defaultTransformConfig)
	podCfg := modifiedConfig["Pod"]
	podCfg.properties = append(podCfg.properties, ExtractProperty{Name: "extra", JSONPath: "{.spec.extra}"})
	modifiedConfig["Pod"] = podCfg
	configMu.Lock()
	mergedTransformConfig = modifiedConfig
	configMu.Unlock()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	var callbackCount int
	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {
		callbackCount++
	})
	cw.lastSeenGeneration = 5

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	fakeWatcher.Delete(newFakeConfigObj(5))
	fakeWatcher.Stop()
	<-done

	assert.Equal(t, int64(0), cw.lastSeenGeneration, "generation should reset on delete")
	assert.Equal(t, 1, callbackCount)
}

func TestProcessEvents_Error(t *testing.T) {
	saveAndRestoreConfigState(t)

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {})

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	fakeWatcher.Error(newFakeConfigObj(1))
	result := <-done

	assert.True(t, result, "processEvents should return true on Error event to trigger reconnect")
}

func TestProcessEvents_ContextCanceled(t *testing.T) {
	saveAndRestoreConfigState(t)

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {})

	fakeWatcher := watch.NewFake()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		done <- cw.processEvents(ctx, fakeWatcher)
	}()

	cancel()
	result := <-done

	assert.False(t, result, "processEvents should return false when context is canceled")
}

func TestStart_FeatureDisabled(t *testing.T) {
	origFeature := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = origFeature }()

	config.Cfg.FeatureConfigurableCollection = false

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {})

	// Start should return immediately when feature is disabled.
	cw.Start(context.Background())
	// If it blocks, the test will time out and fail.
}

func TestStart_ContextCanceled(t *testing.T) {
	saveAndRestoreConfigState(t)
	configMu.Lock()
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
	configMu.Unlock()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme)

	cw := NewConfigWatcher(fakeClient, "test-ns", func(keys []string) {})

	// Cancel the context before Start — it should do the initial reload then exit.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		cw.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Start exited as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not exit after context was canceled")
	}
}
