package transforms

import (
	"context"

	"github.com/stolostron/search-collector/pkg/config"
	v1alpha1 "github.com/stolostron/search-v2-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// mergedTransformConfig contains the merged configuration from defaultTransformConfig plus CollectorConfig CR customizations.
// This is populated by LoadAndMergeConfigurableCollection and used by getTransformConfig.
var mergedTransformConfig map[string]ResourceConfig

// LoadAndMergeConfigurableCollection loads the CollectorConfig resource from the cluster and merges it with defaultTransformConfig.
// The merged result is stored in mergedTransformConfig.
func LoadAndMergeConfigurableCollection() {
	if !config.Cfg.FeatureConfigurableCollection {
		klog.Info("Configurable collection feature is disabled, skipping custom config load")
		// Initialize mergedTransformConfig to a copy of defaultTransformConfig
		mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)
		return
	}

	klog.V(1).Info("Loading configurable collection config from cluster")

	dynamicClient := config.GetDynamicClient()
	loadAndMergeConfigurableCollectionWithClient(dynamicClient)
}

// loadAndMergeConfigurableCollectionWithClient is a helper function that accepts a dynamic client for testability.
func loadAndMergeConfigurableCollectionWithClient(dynamicClient dynamic.Interface) {
	// Start with a deep copy of defaultTransformConfig
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)

	// FUTURE: ACM-20047 watch this for changes and update config dynamically
	unstructuredConfig, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "search.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "collectorconfigs",
	}).Namespace(config.Cfg.PodNamespace).Get(context.Background(), "collector-config", metav1.GetOptions{})

	if err != nil {
		klog.Infof("Could not load collector-config resource: %v. Using default config only", err)
		return
	}

	// Convert unstructured to typed CollectorConfig
	var collectorConfig v1alpha1.CollectorConfig
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredConfig.Object, &collectorConfig); err != nil {
		klog.Warningf("Could not convert collector-config to typed object: %v. Using default config only", err)
		return
	}

	klog.V(1).Info("Found collector-config resource, merging with default config")

	// Get collection rules from spec
	collectionRules := collectorConfig.Spec.CollectionRules
	if len(collectionRules) == 0 {
		klog.Warning("No collectionRules found in collector-config resource")
		return
	}

	// merge each rule from collectionRules with mergedTransformConfig
	for _, rule := range collectionRules {
		// Get field suffix for this rule (defaults to empty string)
		fieldSuffix := rule.FieldSuffix

		// FUTURE: Only Include actions are currently supported
		// Only process Include actions
		if rule.Action != v1alpha1.ActionInclude {
			klog.V(2).Infof("Skipping non-include action. Only \"include\" action supported at this time: %s", rule.Action)
			continue
		}

		// Only process rules that have fields specified
		if len(rule.Fields) == 0 {
			klog.V(2).Info("Skipping Include action without fields specified")
			continue
		}

		apiGroups := rule.ResourceSelector.APIGroups
		kinds := rule.ResourceSelector.Kinds

		if len(kinds) == 0 {
			klog.Warning("collectionRules item missing kinds in resourceSelector, skipping")
			continue
		}

		// validation webhook should ensure there's not >1 apiGroup.kind
		// When fields are specified, there should be exactly one kind and one apiGroup
		if len(kinds) != 1 {
			klog.Warningf("Include action with fields must have exactly 1 kind, found %d. Skipping rule.", len(kinds))
			continue
		}

		if len(apiGroups) != 1 {
			klog.Warningf("Include action with fields must have exactly 1 apiGroup, found %d. Skipping rule.", len(apiGroups))
			continue
		}

		// Extract the single kind and apiGroup
		kind := kinds[0]
		apiGroup := apiGroups[0]

		if kind == "" {
			klog.Warning("Kind is empty, skipping rule")
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
				klog.Warningf("Field missing name or jsonPath for resource %s, skipping", resourceKey)
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
				klog.Warningf("Field name '%s' collides with existing property for resource %s. Skipping this field. Built-in field takes precedence. Consider using fieldSuffix in the CollectionRule.", name, resourceKey)
				continue
			}

			extractProp := ExtractProperty{
				Name:     name,
				JSONPath: field.JSONPath,
				DataType: dataTypeFromCRD(field.Type),
			}

			resourceConfig.properties = append(resourceConfig.properties, extractProp)
			klog.V(2).Infof("Added custom field %s to resource %s", name, resourceKey)
		}

		// Update the merged config (not defaultTransformConfig)
		mergedTransformConfig[resourceKey] = resourceConfig
		klog.V(1).Infof("Merged %d custom fields for resource %s", len(rule.Fields), resourceKey)
	}

	klog.Info("Successfully merged configurable collection config")
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

		dst[key] = ResourceConfig{
			properties:         copiedProperties,
			edges:              copiedEdges,
			extractAnnotations: config.extractAnnotations,
			extractConditions:  config.extractConditions,
		}
	}
	return dst
}
