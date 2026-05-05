// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-v2-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// namespaceFilterCache caches the resolved namespace map with a TTL so that
// informers don't each have to fetch and resolve the CollectorConfig.
type namespaceFilterCache struct {
	mu        sync.RWMutex
	allowed   map[string]bool
	expiresAt time.Time
}

// get returns the cached namespace map, refreshing it if it has expired.
func (c *namespaceFilterCache) get() map[string]bool {
	c.mu.RLock()
	if time.Now().Before(c.expiresAt) {
		defer c.mu.RUnlock()
		return c.allowed
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check: another goroutine may have refreshed while we waited for the write lock.
	if time.Now().Before(c.expiresAt) {
		return c.allowed
	}

	c.allowed = resolveCollectNamespaces()
	c.expiresAt = time.Now().Add(time.Duration(config.Cfg.NSFilterCacheTTLMS) * time.Millisecond)
	return c.allowed
}

// invalidate forces the next get() call to refresh the cache.
func (c *namespaceFilterCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expiresAt = time.Time{} // zero time is always before now
}

var nsFilterCache = &namespaceFilterCache{}

// isNamespaceAllowed checks whether a resource in the given namespace should be collected.
func isNamespaceAllowed(namespace string) bool {
	if !config.Cfg.FeatureConfigurableCollection {
		return true
	}

	if namespace == "" { // cluster-scoped resources don't have namespace
		return true
	}

	allowedNSMap := nsFilterCache.get()
	if allowedNSMap == nil { // error in processing allowedNamespaceMap: collect everywhere
		return true
	}

	_, ok := allowedNSMap[namespace]
	return ok
}

// filterBySelectors lists namespaces filtered by matchLabels and matchExpressions.
func filterBySelectors(kubeClient kubernetes.Interface, nsSelector *v1alpha1.NamespaceSelector) (*v1.NamespaceList, error) {
	// Build a label selector from matchLabels and matchExpressions
	labelSelector := ""
	if len(nsSelector.MatchLabels) > 0 || len(nsSelector.MatchExpressions) > 0 {
		ls := &metav1.LabelSelector{
			MatchLabels:      nsSelector.MatchLabels,
			MatchExpressions: nsSelector.MatchExpressions,
		}
		selector, err := metav1.LabelSelectorAsSelector(ls)
		if err != nil {
			klog.Warningf("Error parsing namespace label selector: %v. Skipping label filtering.", err)
		} else {
			labelSelector = selector.String()
		}
	}

	// List namespaces filtered by labels
	nsList, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		klog.Warningf("Error listing namespaces: %v. Skipping namespace filtering.", err)
		return nil, err
	}

	return nsList, nil
}

// filterByGlobs applies include/exclude filepath glob patterns to narrow a namespace list.
func filterByGlobs(nsList *v1.NamespaceList, nsSelector *v1alpha1.NamespaceSelector) map[string]bool {
	result := make(map[string]bool, 0)
	for _, ns := range nsList.Items {
		name := ns.Name

		// Include filter: if include list is specified, namespace must match at least one pattern
		if len(nsSelector.Include) > 0 {
			matched := false
			for _, pattern := range nsSelector.Include {
				ok, err := filepath.Match(pattern, name)
				if err != nil {
					klog.Warningf("Invalid include glob pattern %q: %v", pattern, err)
					continue
				}
				if ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Exclude filter: if namespace matches any exclude pattern, skip it
		if len(nsSelector.Exclude) > 0 {
			excluded := false
			for _, pattern := range nsSelector.Exclude {
				ok, err := filepath.Match(pattern, name)
				if err != nil {
					klog.Warningf("Invalid exclude glob pattern %q: %v", pattern, err)
					continue
				}
				if ok {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// Add namespace to map that met labels, expressions, includes, and excludes
		result[name] = true
	}
	return result
}

// resolveCollectNamespaces fetches the CollectorConfig and resolves the namespaceSelector
// to a flat map of allowed namespace names. Called by nsCache.get() when the cache has expired.
func resolveCollectNamespaces() map[string]bool {
	if !config.Cfg.FeatureConfigurableCollection {
		return nil
	}

	unstructuredConfig, err := config.GetDynamicClient().Resource(schema.GroupVersionResource{
		Group:    "search.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "collectorconfigs",
	}).Namespace(config.Cfg.PodNamespace).Get(context.Background(), "collector-config", metav1.GetOptions{})
	if err != nil {
		klog.Infof("Could not load collector-config resource: %v. Will collect data from all namespaces.", err)
		return nil
	}

	var collectorConfig v1alpha1.CollectorConfig
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredConfig.Object, &collectorConfig); err != nil {
		klog.Warningf("Could not convert collector-config to typed object: %v. Will collect data from all namespaces.", err)
		return nil
	}

	// No collectNamespaces or namespaceSelector configured — collect everywhere
	if collectorConfig.Spec.CollectNamespaces == nil || collectorConfig.Spec.CollectNamespaces.NamespaceSelector == nil {
		klog.V(2).Info("No collectNamespaces or namespaceSelector fields on collector-config. Will collect data from all namespaces.")
		return nil
	}
	nsSelector := collectorConfig.Spec.CollectNamespaces.NamespaceSelector

	// Nothing specified — collect everywhere
	if len(nsSelector.Include) == 0 && len(nsSelector.Exclude) == 0 &&
		len(nsSelector.MatchLabels) == 0 && len(nsSelector.MatchExpressions) == 0 {
		klog.V(2).Info("No include, exclude, matchLabel, or matchExpression on collector-config. Will collect data from all namespaces.")
		return nil
	}

	// List namespaces filtered by labelSelectors and matchExpressions
	nsList, err := filterBySelectors(config.GetKubeClient(config.GetKubeConfig()), nsSelector)
	if err != nil {
		klog.Warningf("Error listing namespaces by matchLabel or matchExpression: %v. Skipping namespace filtering.", err)
		return nil
	}

	// Filter namespaces by include and exclude namespace globs
	result := filterByGlobs(nsList, nsSelector)

	klog.V(3).Infof("Resolved collectNamespaces to %d namespaces: %v", len(result), result)
	return result
}
