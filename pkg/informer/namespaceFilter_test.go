// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"testing"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-v2-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicFake "k8s.io/client-go/dynamic/fake"
	fakeClient "k8s.io/client-go/kubernetes/fake"
)

func TestFilterByGlobsIncludeNoExclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar], include: [foo], exclude: []
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{"foo"},
		Exclude: []string{},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: foo remains
	assert.Equal(t, 1, len(result))
	assert.Equal(t, true, result["foo"])
}

func TestFilterByGlobsExcludeNoInclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar], include: [], exclude: [foo]
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{},
		Exclude: []string{"foo"},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: bar remains
	assert.Equal(t, 1, len(result))
	assert.Equal(t, true, result["bar"])
}

func TestFilterByGlobsIncludeMatchExclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar], include: [foo], exclude: [foo]
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{"foo"},
		Exclude: []string{"foo"},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: no namespaces to collect
	assert.Equal(t, 0, len(result))
}

func TestFilterByGlobsIncludeExclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar], include: [foo], exclude: [bar]
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{"foo"},
		Exclude: []string{"bar"},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: foo remains
	assert.Equal(t, 1, len(result))
	assert.Equal(t, true, result["foo"])
}

func TestFilterByGlobsIncludeWildcardNoExclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar, baz], include: [ba*], exclude: []
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: v1.ObjectMeta{Name: "baz"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{"ba*"},
		Exclude: []string{},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: bar and baz remain
	assert.Equal(t, 2, len(result))
	assert.Equal(t, true, result["bar"])
	assert.Equal(t, true, result["baz"])
}

func TestFilterByGlobsExcludeWildcardNoInclude(t *testing.T) {
	// Given: a list of namespaces: [foo, bar, baz], include: [], exclude: [ba*]
	namespaceList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
			{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
			{ObjectMeta: v1.ObjectMeta{Name: "baz"}},
		},
	}

	namespaceSelector := &v1alpha1.NamespaceSelector{
		Include: []string{},
		Exclude: []string{"ba*"},
	}

	// When: we filter on namespaceList
	result := filterByGlobs(namespaceList, namespaceSelector)

	// Then: foo
	assert.Equal(t, 1, len(result))
	assert.Equal(t, true, result["foo"])
}

func TestFilterBySelectorsMatchLabels(t *testing.T) {
	// Given a list of namespaces: [prod-app, dev-app, no-labels], selector matching env=prod
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app", Labels: map[string]string{"env": "prod"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "no-labels"}},
	)

	nsSelector := &v1alpha1.NamespaceSelector{
		MatchLabels: map[string]string{"env": "prod"},
	}

	// When: we filter by label selectors
	result, err := filterBySelectors(fc, nsSelector)

	// Then: prod-app remains
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "prod-app", result.Items[0].Name)
}

func TestFilterBySelectorsMatchExpressionsIn(t *testing.T) {
	// Given a list of namespaces: [prod-app, dev-app, no-labels], selector matching env=prod
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app", Labels: map[string]string{"env": "prod"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "no-labels"}},
	)

	nsSelector := &v1alpha1.NamespaceSelector{
		MatchExpressions: []v1.LabelSelectorRequirement{
			{Key: "env", Values: []string{"prod"}, Operator: v1.LabelSelectorOpIn},
		},
	}

	// When: we filter by expression selectors
	result, err := filterBySelectors(fc, nsSelector)

	// Then: prod-app remains
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "prod-app", result.Items[0].Name)
}

func TestFilterBySelectorsMatchExpressionsNotIn(t *testing.T) {
	// Given a list of namespaces: [prod-app, dev-app, no-labels], selector matching env!=prod
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app", Labels: map[string]string{"env": "prod"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "no-labels"}},
	)

	nsSelector := &v1alpha1.NamespaceSelector{
		MatchExpressions: []v1.LabelSelectorRequirement{
			{Key: "env", Values: []string{"prod"}, Operator: v1.LabelSelectorOpNotIn},
		},
	}

	// When: we filter by expression selectors
	result, err := filterBySelectors(fc, nsSelector)

	// Then: dev-app and no-labels remains
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Items))
	assert.Equal(t, "dev-app", result.Items[0].Name)
	assert.Equal(t, "no-labels", result.Items[1].Name)
}

func TestFilterBySelectorsMatchExpressionsNotInWithLabels(t *testing.T) {
	// Given a list of namespaces: [prod-app, dev-app, no-labels], selector matching env!=prod and label env=dev
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app", Labels: map[string]string{"env": "prod"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "no-labels"}},
	)

	nsSelector := &v1alpha1.NamespaceSelector{
		MatchExpressions: []v1.LabelSelectorRequirement{
			{Key: "env", Values: []string{"prod"}, Operator: v1.LabelSelectorOpNotIn},
		},
		MatchLabels: map[string]string{"env": "dev"},
	}

	// When: we filter by expression selectors
	result, err := filterBySelectors(fc, nsSelector)

	// Then: dev-app remains
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Items))
	assert.Equal(t, "dev-app", result.Items[0].Name)
}

func TestFilterBySelectorsThenGlobs(t *testing.T) {
	// Given a list of namespaces: [prod-app, dev-app, dev-app-2, no-labels], selector matching env!=prod and label env=dev and glob Include:[dev-*]
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app", Labels: map[string]string{"env": "prod"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app-2", Labels: map[string]string{"env": "dev"}}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "no-labels"}},
	)

	nsSelector := &v1alpha1.NamespaceSelector{
		MatchExpressions: []v1.LabelSelectorRequirement{
			{Key: "env", Values: []string{"prod"}, Operator: v1.LabelSelectorOpNotIn},
		},
		MatchLabels: map[string]string{"env": "dev"},
		Include:     []string{"dev-*"},
		Exclude:     []string{},
	}

	// When: we filter by expression selectors and glob
	interim, err := filterBySelectors(fc, nsSelector)
	result := filterByGlobs(interim, nsSelector)

	// Then: dev-app and dev-app-2 remain
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, true, result["dev-app"])
	assert.Equal(t, true, result["dev-app-2"])
}

func TestFilterByEmptySelectors(t *testing.T) {
	// Given: a list of namespaces: [foo, bar, baz] with no filters
	fc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "foo"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "bar"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "baz"}},
	)
	namespaceSelector := &v1alpha1.NamespaceSelector{}

	// When: we filter on namespaceList
	interim, err := filterBySelectors(fc, namespaceSelector)
	result := filterByGlobs(interim, namespaceSelector)

	// Then: foo, bar, and baz remain
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, true, result["foo"])
	assert.Equal(t, true, result["bar"])
	assert.Equal(t, true, result["baz"])
}

// setNSFilterCache sets the namespace cache for testing and returns a cleanup function.
func setNSFilterCache(allowed map[string]bool) func() {
	origAllowed := nsFilterCache.allowed
	origExpiry := nsFilterCache.expiresAt
	nsFilterCache.allowed = allowed
	nsFilterCache.expiresAt = time.Now().Add(10 * time.Minute) // won't expire during test
	return func() {
		nsFilterCache.allowed = origAllowed
		nsFilterCache.expiresAt = origExpiry
	}
}

// refreshWith resolves and caches the result.
func TestRefreshWith(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"
	config.Cfg.NSFilterCacheTTLMS = 300000

	dc := fakeDynamicClientWithCollectorConfig(map[string]interface{}{
		"include": []interface{}{"prod-*"},
	})
	kc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app"}},
	)

	c := &namespaceFilterCache{}

	result := c.refreshWith(dc, kc)
	assert.Equal(t, map[string]bool{"prod-app": true}, result)
	assert.True(t, time.Now().Before(c.expiresAt), "expiresAt should be set in the future")
}

// refreshWith skips resolve if another goroutine already refreshed (double-check path).
func TestRefreshWith_DoubleCheck(t *testing.T) {
	config.Cfg.NSFilterCacheTTLMS = 300000

	c := &namespaceFilterCache{
		allowed:   map[string]bool{"already-refreshed": true},
		expiresAt: time.Now().Add(10 * time.Minute),
	}

	// Clients are nil — they should never be used since the cache is still valid.
	result := c.refreshWith(nil, nil)
	assert.Equal(t, map[string]bool{"already-refreshed": true}, result)
}

// regenerate() refreshes cache and resets TTL.
func TestRegenerate(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"
	config.Cfg.NSFilterCacheTTLMS = 300000

	dc := fakeDynamicClientWithCollectorConfig(map[string]interface{}{
		"include": []interface{}{"ns-*"},
	})
	kc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "ns-one"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "ns-two"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "other"}},
	)

	c := &namespaceFilterCache{
		allowed:   map[string]bool{"old": true},
		expiresAt: time.Now().Add(10 * time.Minute),
	}

	c.regenerateWith(dc, kc)

	assert.Equal(t, map[string]bool{"ns-one": true, "ns-two": true}, c.allowed)
	assert.True(t, time.Now().Before(c.expiresAt))
}

// Feature flag disabled: allow everything regardless of cache contents.
func TestIsNamespaceAllowed_FeatureDisabled(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = false

	defer setNSFilterCache(nil)()
	assert.True(t, nsFilterCache.isNamespaceAllowed("any-ns"))

	defer setNSFilterCache(map[string]bool{})()
	assert.True(t, nsFilterCache.isNamespaceAllowed("any-ns"))

	defer setNSFilterCache(map[string]bool{"other": true})()
	assert.True(t, nsFilterCache.isNamespaceAllowed("any-ns"))
}

// Nil map means no filter was configured: allow all namespaces.
func TestIsNamespaceAllowed_NilMap(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(nil)()
	assert.True(t, nsFilterCache.isNamespaceAllowed("any-ns"))
}

// Empty map means selector matched zero namespaces: block everything.
func TestIsNamespaceAllowed_EmptyMap(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{})()
	assert.False(t, nsFilterCache.isNamespaceAllowed("any-ns"))
}

// Cluster-scoped resources (empty namespace) always pass.
func TestIsNamespaceAllowed_ClusterScoped(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true})()
	assert.True(t, nsFilterCache.isNamespaceAllowed(""))

	defer setNSFilterCache(map[string]bool{})()
	assert.True(t, nsFilterCache.isNamespaceAllowed(""))
}

// Namespace in the allowed map passes.
func TestIsNamespaceAllowed_Allowed(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true, "ns2": true})()
	assert.True(t, nsFilterCache.isNamespaceAllowed("ns1"))
	assert.True(t, nsFilterCache.isNamespaceAllowed("ns2"))
}

// Namespace not in the allowed map is blocked.
func TestIsNamespaceAllowed_NotAllowed(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true})()
	assert.False(t, nsFilterCache.isNamespaceAllowed("ns2"))
	assert.False(t, nsFilterCache.isNamespaceAllowed("kube-system"))
}

var collectorConfigGVR = schema.GroupVersionResource{
	Group:    "search.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "collectorconfigs",
}

// Helper to build a fake dynamic client seeded with a CollectorConfig CR.
func fakeDynamicClientWithCollectorConfig(nsSelector map[string]interface{}) *dynamicFake.FakeDynamicClient {
	scheme := runtime.NewScheme()

	spec := map[string]interface{}{}
	if nsSelector != nil {
		spec["collectNamespaces"] = map[string]interface{}{
			"namespaceSelector": nsSelector,
		}
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "search.open-cluster-management.io/v1alpha1",
			"kind":       "CollectorConfig",
			"metadata": map[string]interface{}{
				"name":      "collector-config",
				"namespace": "open-cluster-management",
			},
			"spec": spec,
		},
	}

	return dynamicFake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			collectorConfigGVR: "CollectorConfigList",
		},
		obj,
	)
}

// Feature disabled: returns nil.
func TestResolveCollectNamespaces_FeatureDisabled(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = false

	result := resolveCollectNamespaces(nil, nil)
	assert.Nil(t, result)
}

// CollectorConfig not found: returns nil (collect everywhere).
func TestResolveCollectNamespaces_ConfigNotFound(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"

	// Empty dynamic client — no CollectorConfig exists
	scheme := runtime.NewScheme()
	dc := dynamicFake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			collectorConfigGVR: "CollectorConfigList",
		},
	)

	result := resolveCollectNamespaces(dc, nil)
	assert.Nil(t, result)
}

// CollectorConfig with no namespaceSelector: returns nil (collect everywhere).
func TestResolveCollectNamespaces_NoNamespaceSelector(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"

	dc := fakeDynamicClientWithCollectorConfig(nil)

	result := resolveCollectNamespaces(dc, nil)
	assert.Nil(t, result)
}

// CollectorConfig with empty namespaceSelector: returns nil (collect everywhere).
func TestResolveCollectNamespaces_EmptySelector(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"

	dc := fakeDynamicClientWithCollectorConfig(map[string]interface{}{})

	result := resolveCollectNamespaces(dc, nil)
	assert.Nil(t, result)
}

// CollectorConfig with include glob: resolves filtered namespaces.
func TestResolveCollectNamespaces_WithInclude(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	origNS := config.Cfg.PodNamespace
	defer func() { config.Cfg.PodNamespace = origNS }()
	config.Cfg.PodNamespace = "open-cluster-management"

	dc := fakeDynamicClientWithCollectorConfig(map[string]interface{}{
		"include": []interface{}{"prod-*"},
	})
	kc := fakeClient.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-app"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "prod-db"}},
		&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "dev-app"}},
	)

	result := resolveCollectNamespaces(dc, kc)
	assert.Equal(t, 2, len(result))
	assert.True(t, result["prod-app"])
	assert.True(t, result["prod-db"])
}
