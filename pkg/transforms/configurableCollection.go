package transforms

import (
	"context"
	"fmt"
	"strings"

	"github.com/stolostron/search-collector/pkg/config"
	v1alpha1 "github.com/stolostron/search-v2-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// mergedTransformConfig contains the merged configuration from defaultTransformConfig plus CollectorConfig CR customizations.
// This is populated by LoadAndMergeConfigurableCollection and used by getTransformConfig.
// Wildcard entries like "*" (core group) or "*.apps" are used for apigroup-wide collectConditions.
var mergedTransformConfig map[string]ResourceConfig

// excludedResources holds the set of resource keys that should not be collected.
// Keys follow the same scheme as mergedTransformConfig: "Kind.apiGroup" for specific kinds,
// "*" for all core-group resources, "*.apps" for all resources in the apps apiGroup.
// Populated by mergeExcludeRules during config load; checked at the informer layer.
//
// Thread safety: currently written once at startup (before informers start) and read
// concurrently by informer goroutines. Safe for now because reads begin only after
// LoadAndMergeConfigurableCollection returns. When ACM-20047 adds dynamic reload,
// this will need a sync.RWMutex — see nsFilterCache for the expected pattern.
var excludedResources map[string]struct{}

// LoadAndMergeConfigurableCollection loads the CollectorConfig resource from the cluster and merges it with defaultTransformConfig.
// The merged result is stored in mergedTransformConfig.
func LoadAndMergeConfigurableCollection() {
	if !config.Cfg.FeatureConfigurableCollection {
		klog.Info("Configurable collection feature is disabled, skipping custom config load")
		mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
		excludedResources = map[string]struct{}{}
		return
	}

	klog.V(1).Info("Loading configurable collection config from cluster")

	dynamicClient := config.GetDynamicClient()
	loadAndMergeConfigurableCollectionWithClient(dynamicClient)
}

// Status condition constants for the CollectorConfig CR.
// These mirror the constants defined in the search-v2-operator API types.
const (
	collectorConfigConditionApplied   = "Applied"
	collectorConfigReasonApplied      = "Applied"
	collectorConfigReasonRulesSkipped = "RulesSkipped"
	collectorConfigReasonLoadError    = "LoadError"
)

// collectorConfigGVR is the GroupVersionResource for CollectorConfig.
var collectorConfigGVR = schema.GroupVersionResource{
	Group:    "search.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "collectorconfigs",
}

// loadAndMergeConfigurableCollectionWithClient is a helper function that accepts a dynamic client for testability.
func loadAndMergeConfigurableCollectionWithClient(dynamicClient dynamic.Interface) {
	// Start with a deep copy of defaultTransformConfig and a fresh exclude set.
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
	excludedResources = map[string]struct{}{}

	namespace := config.Cfg.PodNamespace

	// FUTURE: ACM-20047 watch this for changes and update config dynamically
	configObj, err := dynamicClient.Resource(collectorConfigGVR).
		Namespace(namespace).
		Get(context.Background(), "merged-collector-config", metav1.GetOptions{})

	if err != nil {
		// CR not found or not accessible — no status to update, just log and return.
		klog.Infof("Could not load merged-collector-config resource: %v. Using default config only", err)
		return
	}

	// Convert unstructured to typed CollectorConfig
	var collectorConfig v1alpha1.CollectorConfig
	if convErr := runtime.DefaultUnstructuredConverter.FromUnstructured(configObj.Object, &collectorConfig); convErr != nil {
		msg := fmt.Sprintf("Could not convert merged-collector-config to typed object: %v. Using default config only", convErr)
		klog.Warning(msg)
		updateCollectorConfigStatus(dynamicClient, namespace, configObj, []string{msg}, collectorConfigReasonLoadError)
		return
	}

	klog.V(1).Info("Found merged-collector-config resource, merging with default config")

	// Get collection rules from spec
	collectionRules := collectorConfig.Spec.CollectionRules
	if len(collectionRules) == 0 {
		klog.Warning("No collectionRules found in merged-collector-config resource")
		// Empty rules is a valid (though unusual) configuration — mark as Applied.
		updateCollectorConfigStatus(dynamicClient, namespace, configObj, nil, collectorConfigReasonApplied)
		return
	}

	// warnings accumulates messages for rules or fields that were skipped.
	// These become the status condition message so users can see issues via `oc describe`.
	var warnings []string

	// merge each rule from collectionRules with mergedTransformConfig
	for _, rule := range collectionRules {
		// Get field suffix for this rule (defaults to empty string)
		fieldSuffix := rule.FieldSuffix

		if rule.Action == v1alpha1.ActionExclude {
			mergeExcludeRules(rule.ResourceSelector.APIGroups, rule.ResourceSelector.Kinds)
			continue
		}

		hasFields := len(rule.Fields) > 0
		hasCollectConditions := rule.CollectConditions != nil && *rule.CollectConditions
		hasCollectAnnotations := rule.CollectAnnotations != nil && *rule.CollectAnnotations
		hasCollectPrinterColumns := rule.CollectAdditionalPrinterColumnsPriority != nil

		// Only process rules that have actionable configuration — check BEFORE unmerging
		// any prior exclude, so a malformed include rule does not silently cancel an exclude.
		if !hasFields && !hasCollectConditions && !hasCollectPrinterColumns && !hasCollectAnnotations {
			msg := "Rule skipped: include action requires at least one field, collectConditions, collectAnnotations, or collectAdditionalPrinterColumnsPriority"
			klog.Warning("Skipping collection rule. Include action without fields, collectConditions, collectAnnotations, or collectAdditionalPrinterColumnsPriority specified.")
			warnings = append(warnings, msg)
			continue
		}

		// "Last entry wins": a valid include rule cancels any prior exclude for the same resource.
		// Uses the same key scheme as mergeExcludeRules so exact keys are matched and removed.
		// Note: wildcard-vs-specific mismatches (e.g. exclude "*.*" followed by include
		// "Deployment.apps") are not resolved — those are documented as a known limitation.
		unmergeExcludeRules(rule.ResourceSelector.APIGroups, rule.ResourceSelector.Kinds)

		apiGroups := rule.ResourceSelector.APIGroups
		kinds := rule.ResourceSelector.Kinds

		// Process collectConditions
		if hasCollectConditions {
			mergeCollectConditions(apiGroups, kinds)
		}

		// Process collectAnnotations
		if hasCollectAnnotations {
			mergeCollectAnnotations(apiGroups, kinds)
		}

		// Process collectAdditionalPrinterColumnsPriority
		if hasCollectPrinterColumns {
			mergeCollectPrinterColumns(apiGroups, kinds, *rule.CollectAdditionalPrinterColumnsPriority)
		}

		if !hasFields {
			continue
		}

		// Filter out wildcard kinds before fields processing — fields require a specific kind.
		specificKinds := make([]string, 0, len(kinds))
		for _, k := range kinds {
			if k != "*" {
				specificKinds = append(specificKinds, k)
			}
		}
		kinds = specificKinds

		if len(kinds) == 0 {
			msg := "Rule skipped: resourceSelector is missing kinds"
			klog.Warning("Skipping collection rule. Item missing kinds in resourceSelector.")
			warnings = append(warnings, msg)
			continue
		}

		// validation webhook should ensure there's not >1 apiGroup.kind
		// When fields are specified, there should be exactly one kind and one apiGroup
		if len(kinds) != 1 {
			msg := fmt.Sprintf("Rule skipped: include action with fields must specify exactly 1 kind, found %d", len(kinds))
			klog.Warningf("Skipping collection rule. Include action with fields must have exactly 1 kind, found %d.", len(kinds))
			warnings = append(warnings, msg)
			continue
		}

		if len(apiGroups) != 1 {
			msg := fmt.Sprintf("Rule skipped: include action with fields must specify exactly 1 apiGroup, found %d", len(apiGroups))
			klog.Warningf("Skipping collection rule. Include action with fields must have exactly 1 apiGroup, found %d.", len(apiGroups))
			warnings = append(warnings, msg)
			continue
		}

		// Extract the single kind and apiGroup
		kind := kinds[0]
		apiGroup := apiGroups[0]

		if kind == "" {
			msg := "Rule skipped: kind is empty"
			klog.Warning("Skipping collection rule. Kind is empty.")
			warnings = append(warnings, msg)
			continue
		}

		resourceKey := kind
		if apiGroup != "" {
			resourceKey = kind + "." + apiGroup
		}

		// get existing key for kind.apiGroup resource from merged config
		resourceConfig, exists := mergedTransformConfig[resourceKey]
		if !exists {
			resourceConfig = ResourceConfig{
				properties: []ExtractProperty{},
			}
		}

		// parse and add new fields to resourceConfig
		for _, field := range rule.Fields {
			if field.Name == "" || field.JSONPath == "" {
				msg := fmt.Sprintf("Field skipped for %s: name or jsonPath is empty", resourceKey)
				klog.Warningf("Skipping collection rule. Field missing name or jsonPath for resource %s.", resourceKey)
				warnings = append(warnings, msg)
				continue
			}

			// Apply field suffix (if configured) to avoid collisions
			// User provides suffix without dot (e.g., "grc"), we prepend the dot to get "field.grc"
			name := field.Name
			if fieldSuffix != "" {
				name = name + "." + fieldSuffix
			}

			// Check for collision with existing properties in the resource config
			collision := false
			for _, existingProp := range resourceConfig.properties {
				if existingProp.Name == name {
					collision = true
					break
				}
			}

			if collision {
				msg := fmt.Sprintf("Field %q skipped for %s: collides with a built-in field. Use fieldSuffix to avoid collisions", name, resourceKey)
				klog.Warningf("Skipping collection rule. Field name '%s' collides with existing property for resource %s. Built-in field takes precedence. Consider using fieldSuffix in the CollectionRule.", name, resourceKey)
				warnings = append(warnings, msg)
				continue
			}

			// Normalize the jsonPath: the k8s jsonpath library requires expressions to be
			// wrapped in curly braces (e.g. "{.spec.myField}"), but users unfamiliar with
			// this library convention may omit them. Strip any partial braces and
			// re-wrap consistently so ".spec.myField", "{.spec.myField}", and
			// malformed inputs like "{.spec.myField" all produce "{.spec.myField}".
			jsonPath := "{" + strings.TrimSuffix(strings.TrimPrefix(field.JSONPath, "{"), "}") + "}"

			extractProp := ExtractProperty{
				Name:     name,
				JSONPath: jsonPath,
				DataType: dataTypeFromCRD(field.Type),
			}

			resourceConfig.properties = append(resourceConfig.properties, extractProp)
			klog.V(2).Infof("Added custom field %s to resource %s", name, resourceKey)
		}

		// Update the merged config (not defaultTransformConfig)
		mergedTransformConfig[resourceKey] = resourceConfig
		klog.V(1).Infof("Merged %d custom fields for resource %s", len(rule.Fields), resourceKey)
	}

	// Determine final condition reason based on whether any rules were skipped.
	reason := collectorConfigReasonApplied
	if len(warnings) > 0 {
		reason = collectorConfigReasonRulesSkipped
	}
	updateCollectorConfigStatus(dynamicClient, namespace, configObj, warnings, reason)

	klog.Info("Successfully merged configurable collection config")
}

// updateCollectorConfigStatus writes an "Applied" status condition to the CollectorConfig CR.
// warnings contains human-readable messages for any rules or fields that were skipped.
// lastTransitionTime is only updated when the condition status (True/False) changes, following
// the Kubernetes convention that lastTransitionTime reflects the last state *transition*, not
// the last time the condition was evaluated.
// This is best-effort: failures are logged but do not abort the collector.
func updateCollectorConfigStatus(dynamicClient dynamic.Interface, namespace string,
	configObj *unstructured.Unstructured, warnings []string, reason string) {

	// maxStatusWarnings is the maximum number of individual warning messages to include
	// in the condition Message before truncating with "... and N more". This keeps the
	// message readable while still surfacing the most actionable errors first.
	const maxStatusWarnings = 3

	conditionStatus := metav1.ConditionTrue
	message := "Configuration applied successfully."
	if len(warnings) > 0 {
		conditionStatus = metav1.ConditionFalse
		if len(warnings) > maxStatusWarnings {
			message = strings.Join(warnings[:maxStatusWarnings], "; ") +
				fmt.Sprintf("; ... and %d more", len(warnings)-maxStatusWarnings)
		} else {
			message = strings.Join(warnings, "; ")
		}
	}

	// Preserve lastTransitionTime if the condition status hasn't changed.
	// Search by type (not by index) so other conditions don't affect the lookup.
	// Only update the timestamp on True↔False transitions per Kubernetes conventions.
	lastTransitionTime := metav1.Now().UTC().Format("2006-01-02T15:04:05Z")
	existingStatus, _ := configObj.Object["status"].(map[string]interface{})
	if existingStatus == nil {
		existingStatus = map[string]interface{}{}
	}
	if conditions, ok := existingStatus["conditions"].([]interface{}); ok {
		for _, c := range conditions {
			if cond, ok := c.(map[string]interface{}); ok &&
				cond["type"] == collectorConfigConditionApplied &&
				cond["status"] == string(conditionStatus) {
				if t, ok := cond["lastTransitionTime"].(string); ok && t != "" {
					lastTransitionTime = t
				}
				break
			}
		}
	}

	condition := map[string]interface{}{
		"type":               collectorConfigConditionApplied,
		"status":             string(conditionStatus),
		"reason":             reason,
		"message":            message,
		"lastTransitionTime": lastTransitionTime,
	}

	// Upsert the Applied condition — replace it if it already exists, otherwise append.
	// This preserves any other conditions or status fields already present on the CR.
	existingConditions, _ := existingStatus["conditions"].([]interface{})
	upserted := false
	for i, c := range existingConditions {
		if cond, ok := c.(map[string]interface{}); ok && cond["type"] == collectorConfigConditionApplied {
			existingConditions[i] = condition
			upserted = true
			break
		}
	}
	if !upserted {
		existingConditions = append(existingConditions, condition)
	}
	existingStatus["conditions"] = existingConditions
	configObj.Object["status"] = existingStatus

	if _, err := dynamicClient.Resource(collectorConfigGVR).Namespace(namespace).
		Update(context.Background(), configObj, metav1.UpdateOptions{}, "status"); err != nil {
		klog.Warningf("Could not update CollectorConfig status conditions: %v", err)
		return
	}
	klog.V(2).Infof("Updated CollectorConfig status: Applied=%s reason=%s", conditionStatus, reason)
}

// mergeExcludeRules records the given apiGroups+kinds in excludedResources so they are
// skipped at the informer layer. Follows the same key scheme as mergedTransformConfig:
// - specific kind:  "Lease.coordination.k8s.io"
// - all kinds in group:  "*.coordination.k8s.io"
// - all core-group kinds: "*"
func mergeExcludeRules(apiGroups, kinds []string) {
	for _, apiGroup := range apiGroups {
		for _, kind := range kinds {
			if kind == "" {
				continue
			}
			resourceKey := kind
			if apiGroup != "" {
				resourceKey = kind + "." + apiGroup
			}
			excludedResources[resourceKey] = struct{}{}
			klog.V(2).Infof("Excluding resource from collection: %s", resourceKey)
		}
	}
}

// unmergeExcludeRules removes the given apiGroups+kinds from excludedResources.
// Called when an include rule follows an exclude for the same resource so that the
// include wins ("last entry wins" semantics within a single CollectorConfig).
func unmergeExcludeRules(apiGroups, kinds []string) {
	for _, apiGroup := range apiGroups {
		for _, kind := range kinds {
			if kind == "" {
				continue
			}
			resourceKey := kind
			if apiGroup != "" {
				resourceKey = kind + "." + apiGroup
			}
			delete(excludedResources, resourceKey)
			klog.V(2).Infof("Include rule cancels prior exclude for resource %s", resourceKey)
		}
	}
}

// IsResourceExcluded reports whether a resource should be excluded from collection.
// Checks in order:
//  1. Exact match:          "Lease.coordination.k8s.io" / "Event" (core group)
//  2. Kind wildcard:        "*.coordination.k8s.io"     / "*" (core group)
//  3. Group wildcard:       "Lease.*"  (any apiGroup for this kind) — not common, but consistent
//  4. Global wildcard:      "*.*"      (all kinds in all groups)
func IsResourceExcluded(group, kind string) bool {
	if excludedResources == nil {
		return false
	}
	// 1. Exact match
	exactKey := kind
	if group != "" {
		exactKey = kind + "." + group
	}
	if _, ok := excludedResources[exactKey]; ok {
		return true
	}
	// 2. Wildcard kind for this group: "*.group" or "*" (core)
	kindWildcard := "*"
	if group != "" {
		kindWildcard = "*." + group
	}
	if _, ok := excludedResources[kindWildcard]; ok {
		return true
	}
	// 3. This kind in any group: "Kind.*" (matches even core-group resources when apiGroups: ["*"])
	if _, ok := excludedResources[kind+".*"]; ok {
		return true
	}
	// 4. Global wildcard: "*.*" (all kinds across all apiGroups)
	_, ok := excludedResources["*.*"]
	return ok
}

// mergeCollectConditions enables condition extraction for the given apiGroups and kinds.
// When kind is "*", a wildcard entry (e.g., "*" or "*.apps") is stored in mergedTransformConfig,
// enabling condition extraction for all resources in that apiGroup at runtime.
// For specific kinds, extractConditions is set for each kind+apiGroup combination.
func mergeCollectConditions(apiGroups, kinds []string) {
	for _, apiGroup := range apiGroups {
		for _, kind := range kinds {
			if kind == "" {
				continue
			}
			resourceKey := kind
			if apiGroup != "" {
				resourceKey = kind + "." + apiGroup
			}
			resourceConfig, exists := mergedTransformConfig[resourceKey]
			if !exists {
				resourceConfig = ResourceConfig{
					properties: []ExtractProperty{},
				}
			}
			resourceConfig.extractConditions = true
			mergedTransformConfig[resourceKey] = resourceConfig
			klog.V(2).Infof("Enabled condition collection for resource %s", resourceKey)
		}
	}
}

// mergeCollectAnnotations enables annotation extraction for the given apiGroups and kinds.
// When kind is "*", a wildcard entry (e.g., "*" or "*.apps") is stored in mergedTransformConfig,
// enabling annotation extraction for all resources in that apiGroup at runtime.
// For specific kinds, extractAnnotations is set for each kind+apiGroup combination.
func mergeCollectAnnotations(apiGroups, kinds []string) {
	for _, apiGroup := range apiGroups {
		for _, kind := range kinds {
			if kind == "" {
				continue
			}
			resourceKey := kind
			if apiGroup != "" {
				resourceKey = kind + "." + apiGroup
			}
			resourceConfig, exists := mergedTransformConfig[resourceKey]
			if !exists {
				resourceConfig = ResourceConfig{
					properties: []ExtractProperty{},
				}
			}
			resourceConfig.extractAnnotations = true
			mergedTransformConfig[resourceKey] = resourceConfig
			klog.V(2).Infof("Enabled annotation collection for resource %s", resourceKey)
		}
	}
}

// mergeCollectPrinterColumns sets the additionalPrinterColumns priority threshold for the given apiGroups and kinds.
// When kind is "*", a wildcard entry (e.g., "*" or "*.apps") is stored in mergedTransformConfig,
// enabling printer column collection for all resources in that apiGroup at runtime.
// For specific kinds, the priority is set for each kind+apiGroup combination.
func mergeCollectPrinterColumns(apiGroups, kinds []string, priority int) {
	for _, apiGroup := range apiGroups {
		for _, kind := range kinds {
			if kind == "" {
				continue
			}
			resourceKey := kind
			if apiGroup != "" {
				resourceKey = kind + "." + apiGroup
			}
			resourceConfig, exists := mergedTransformConfig[resourceKey]
			if !exists {
				resourceConfig = ResourceConfig{
					properties: []ExtractProperty{},
				}
			}
			// Take the max priority — higher values are more permissive (collect more columns).
			// This prevents a later rule from accidentally narrowing an earlier rule's threshold.
			if resourceConfig.additionalPrinterColumnsPriority == nil || priority > *resourceConfig.additionalPrinterColumnsPriority {
				resourceConfig.additionalPrinterColumnsPriority = &priority
			}
			mergedTransformConfig[resourceKey] = resourceConfig
			klog.V(2).Infof("Set additionalPrinterColumns priority to %d for resource %s", *resourceConfig.additionalPrinterColumnsPriority, resourceKey)
		}
	}
}

// dataTypeFromCRD converts v1alpha1.DataType to internal DataType
func dataTypeFromCRD(crdType v1alpha1.DataType) DataType {
	switch crdType {
	case v1alpha1.DataTypeBytes:
		return DataTypeBytes
	case v1alpha1.DataTypeSlice:
		return DataTypeSlice
	case v1alpha1.DataTypeString:
		return DataTypeString
	case v1alpha1.DataTypeNumber:
		return DataTypeNumber
	case v1alpha1.DataTypeMapString:
		return DataTypeMapString
	default:
		return DataTypeString
	}
}

// deepCopyTransformConfig creates a deep copy of the transform config map.
func deepCopyTransformConfig(src map[string]ResourceConfig) map[string]ResourceConfig {
	dst := make(map[string]ResourceConfig, len(src))
	for key, config := range src {
		// Deep copy the properties slice
		copiedProperties := make([]ExtractProperty, len(config.properties))
		copy(copiedProperties, config.properties)

		// Deep copy the edges slice
		copiedEdges := make([]ExtractEdge, len(config.edges))
		copy(copiedEdges, config.edges)

		var copiedPriority *int
		if config.additionalPrinterColumnsPriority != nil {
			p := *config.additionalPrinterColumnsPriority
			copiedPriority = &p
		}

		dst[key] = ResourceConfig{
			properties:                       copiedProperties,
			edges:                            copiedEdges,
			extractAnnotations:               config.extractAnnotations,
			extractConditions:                config.extractConditions,
			additionalPrinterColumnsPriority: copiedPriority,
		}
	}
	return dst
}
