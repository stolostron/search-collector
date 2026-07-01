package transforms

import (
	"context"
	"reflect"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// ConfigWatcher watches the merged-collector-config CollectorConfig CR and hot-reloads
// the mergedTransformConfig when it changes. After reloading, it triggers targeted
// informer resyncs for resource types whose config actually changed.
type ConfigWatcher struct {
	dynamicClient      dynamic.Interface
	namespace          string
	resyncCallback     func(keys []string) // called with affected config keys after a reload
	lastSeenGeneration int64               // tracks metadata.generation to skip status-only updates
}

// NewConfigWatcher creates a new ConfigWatcher. The resyncCallback is invoked with the
// list of changed config keys after each successful reload, allowing the caller to
// trigger informer resyncs without creating an import cycle.
func NewConfigWatcher(client dynamic.Interface, namespace string, resyncCallback func(keys []string)) *ConfigWatcher {
	return &ConfigWatcher{
		dynamicClient:  client,
		namespace:      namespace,
		resyncCallback: resyncCallback,
	}
}

// Start begins watching merged-collector-config for changes. It performs an initial read
// to close the race window between LoadAndMergeConfigurableCollection() and watch establishment,
// then enters a watch loop with reconnect-on-error and exponential backoff.
// Blocks until ctx is canceled.
func (cw *ConfigWatcher) Start(ctx context.Context) {
	if !config.Cfg.FeatureConfigurableCollection {
		klog.V(2).Info("Configurable collection feature is disabled, config watcher will not start")
		return
	}

	klog.Info("Starting config watcher for merged-collector-config")

	// Initial read to catch any config change that occurred between
	// LoadAndMergeConfigurableCollection() and now.
	cw.reloadAndResync()

	var retries int64
	for {
		select {
		case <-ctx.Done():
			klog.Info("Config watcher stopped")
			return
		default:
		}

		if retries > 0 {
			wait := time.Duration(min(retries*2, 120)) * time.Second
			klog.V(3).Infof("Config watcher: waiting %s before reconnecting", wait)

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				klog.Info("Config watcher stopped during backoff")
				return
			}
		}

		// Re-read current state before establishing watch, in case we missed events
		// during a disconnection.
		if retries > 0 {
			cw.reloadAndResync()
		}

		watcher, err := cw.dynamicClient.Resource(collectorConfigGVR).
			Namespace(cw.namespace).
			Watch(ctx, metav1.ListOptions{
				FieldSelector: "metadata.name=merged-collector-config",
			})
		if err != nil {
			klog.Warningf("Config watcher: error establishing watch: %v", err)
			retries++
			continue
		}

		retries = 0
		klog.V(2).Info("Config watcher: watch established for merged-collector-config")

		// Process watch events until the watch ends or context is canceled.
		watchEnded := cw.processEvents(ctx, watcher)
		watcher.Stop()

		if !watchEnded {
			// Context was canceled
			return
		}
		// Watch ended (timeout or error) — loop will reconnect with backoff
		retries++
	}
}

// processEvents handles watch events. Returns true if the watch ended (needs reconnect),
// false if the context was canceled.
func (cw *ConfigWatcher) processEvents(ctx context.Context, watcher watch.Interface) bool {
	for {
		select {
		case <-ctx.Done():
			return false

		case event, ok := <-watcher.ResultChan():
			if !ok {
				klog.V(2).Info("Config watcher: watch channel closed, will reconnect")
				return true
			}

			switch event.Type {
			case watch.Modified:
				obj, ok := event.Object.(*unstructured.Unstructured)
				if ok {
					gen := obj.GetGeneration()
					if gen == cw.lastSeenGeneration {
						klog.V(3).Info("Config watcher: status-only update (generation unchanged), skipping reload")
						continue
					}
					cw.lastSeenGeneration = gen
				}
				klog.Info("Config watcher: merged-collector-config modified, reloading")
				cw.reloadAndResync()

			case watch.Deleted:
				klog.Warning("Config watcher: merged-collector-config deleted — reverting to defaults. " +
					"This is unexpected; the operator should recreate it.")
				cw.lastSeenGeneration = 0
				cw.reloadAndResync()

			case watch.Added:
				obj, ok := event.Object.(*unstructured.Unstructured)
				if ok {
					cw.lastSeenGeneration = obj.GetGeneration()
				}
				klog.Info("Config watcher: merged-collector-config created, loading config")
				cw.reloadAndResync()

			case watch.Error:
				klog.Warningf("Config watcher: received ERROR event: %v", event.Object)
				return true

			default:
				klog.V(3).Infof("Config watcher: ignoring event type %s", event.Type)
			}
		}
	}
}

// reloadAndResync snapshots the current config, reloads from the cluster, diffs the
// old and new configs, and triggers targeted informer resyncs for affected resource types.
func (cw *ConfigWatcher) reloadAndResync() {
	// Snapshot the current config before reload.
	oldConfig := snapshotConfig()

	// Reload config from cluster (performs atomic swap under configMu.Lock).
	loadAndMergeConfigurableCollectionWithClient(cw.dynamicClient)

	// Snapshot the new config after reload.
	newConfig := snapshotConfig()

	// Diff to find affected config keys.
	affected := diffConfigs(oldConfig, newConfig)
	if len(affected) == 0 {
		klog.V(2).Info("Config watcher: no config changes detected after reload")
		return
	}

	klog.Infof("Config watcher: %d resource config(s) changed, triggering resync: %v", len(affected), affected)
	cw.resyncCallback(affected)
}

// snapshotConfig returns a deep copy of the current mergedTransformConfig.
// The copy is taken under RLock to ensure a consistent snapshot.
func snapshotConfig() map[string]ResourceConfig {
	configMu.RLock()
	snapshot := deepCopyTransformConfig(mergedTransformConfig)
	configMu.RUnlock()
	return snapshot
}

// diffConfigs compares old and new config maps, returning the list of config keys
// that were added, removed, or changed.
func diffConfigs(old, new map[string]ResourceConfig) []string {
	affected := map[string]bool{}

	// Keys added or removed
	for k := range new {
		if _, ok := old[k]; !ok {
			affected[k] = true
		}
	}
	for k := range old {
		if _, ok := new[k]; !ok {
			affected[k] = true
		}
	}

	// Keys with changed config
	for k, newCfg := range new {
		if oldCfg, ok := old[k]; ok && !reflect.DeepEqual(oldCfg, newCfg) {
			affected[k] = true
		}
	}

	result := make([]string, 0, len(affected))
	for k := range affected {
		result = append(result, k)
	}
	return result
}
