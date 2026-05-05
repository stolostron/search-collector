// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-v2-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeClient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
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

// Feature flag disabled: allow everything regardless of cache contents.
func TestIsNamespaceAllowed_FeatureDisabled(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = false

	defer setNSFilterCache(nil)()
	assert.True(t, isNamespaceAllowed("any-ns"))

	defer setNSFilterCache(map[string]bool{})()
	assert.True(t, isNamespaceAllowed("any-ns"))

	defer setNSFilterCache(map[string]bool{"other": true})()
	assert.True(t, isNamespaceAllowed("any-ns"))
}

// Nil map means no filter was configured: allow all namespaces.
func TestIsNamespaceAllowed_NilMap(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(nil)()
	assert.True(t, isNamespaceAllowed("any-ns"))
}

// Empty map means selector matched zero namespaces: block everything.
func TestIsNamespaceAllowed_EmptyMap(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{})()
	assert.False(t, isNamespaceAllowed("any-ns"))
}

// Cluster-scoped resources (empty namespace) always pass.
func TestIsNamespaceAllowed_ClusterScoped(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true})()
	assert.True(t, isNamespaceAllowed(""))

	defer setNSFilterCache(map[string]bool{})()
	assert.True(t, isNamespaceAllowed(""))
}

// Namespace in the allowed map passes.
func TestIsNamespaceAllowed_Allowed(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true, "ns2": true})()
	assert.True(t, isNamespaceAllowed("ns1"))
	assert.True(t, isNamespaceAllowed("ns2"))
}

// Namespace not in the allowed map is blocked.
func TestIsNamespaceAllowed_NotAllowed(t *testing.T) {
	original := config.Cfg.FeatureConfigurableCollection
	defer func() { config.Cfg.FeatureConfigurableCollection = original }()
	config.Cfg.FeatureConfigurableCollection = true

	defer setNSFilterCache(map[string]bool{"ns1": true})()
	assert.False(t, isNamespaceAllowed("ns2"))
	assert.False(t, isNamespaceAllowed("kube-system"))
}
