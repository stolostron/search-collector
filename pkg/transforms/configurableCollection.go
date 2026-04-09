package transforms

import (
	"context"

	"github.com/stolostron/search-collector/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	klog.Info("Loading configurable collection config from cluster")

	dynamicClient := config.GetDynamicClient()
	loadAndMergeConfigurableCollectionWithClient(dynamicClient)
}

// loadAndMergeConfigurableCollectionWithClient is a helper function that accepts a dynamic client for testability.
func loadAndMergeConfigurableCollectionWithClient(dynamicClient dynamic.Interface) {
	// Start with a deep copy of defaultTransformConfig
	mergedTransformConfig = deepCopyTransformConfig(defaultTransformConfig)

	// FUTURE: ACM-20047 watch this for changes and update config dynamically
	gvr := schema.GroupVersionResource{
		Group:    "search.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "collectorconfigs",
	}

	resource, err := dynamicClient.Resource(gvr).Namespace(config.Cfg.PodNamespace).
		Get(context.Background(), "collector-config", metav1.GetOptions{})

	if err != nil {
		klog.Warningf("Could not load collector-config resource: %v. Using default config only", err)
		return
	}

	klog.Info("Found collector-config resource, merging with default config")

	// get spec field from resource
	spec, specFound, _ := unstructuredNested(resource.Object, "spec")
	if !specFound {
		klog.Warning("No spec found in collector-config resource. Using default config only")
		return
	}

	specMap, ok := spec.(map[string]interface{})
	if !ok {
		klog.Warning("spec is not a map in collector-config resource. Using default config only")
		return
	}

	// get collectionRules from spec
	collectionRules, rulesFound, _ := unstructuredNested(specMap, "collectionRules")
	if !rulesFound {
		klog.Info("No collectionRules found in collector-config resource")
		return
	}

	rulesArray, ok := collectionRules.([]interface{})
	if !ok {
		klog.Warning("collectionRules is not an array in collector-config resource. Using default config only")
		return
	}

	// merge each rule from collectionRules with mergedTransformConfig
	for _, rule := range rulesArray {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			klog.Warning("collectionRules item is not a map, skipping")
			continue
		}

		// FUTURE: Only Include actions are currently supported
		// Only process Include actions
		action, _ := ruleMap["action"].(string)
		if action != "include" {
			klog.V(2).Infof("Skipping non-include action. Only \"include\" action supported at this time: %s", action)
			continue
		}

		// Only process rules that have fields specified
		fields, hasFields := ruleMap["fields"].([]interface{})
		if !hasFields || len(fields) == 0 {
			klog.V(2).Info("Skipping Include action without fields specified")
			continue
		}

		// Extract resourceSelector
		resourceSelector, hasSelectorMap := ruleMap["resourceSelector"].(map[string]interface{})
		if !hasSelectorMap {
			klog.Warning("collectionRules item missing resourceSelector, skipping")
			continue
		}

		apiGroups, _ := resourceSelector["apiGroups"].([]interface{})
		kinds, _ := resourceSelector["kinds"].([]interface{})

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

		// Extract the single kind
		kind, ok := kinds[0].(string)
		if !ok || kind == "" {
			klog.Warning("Kind is not a valid string, skipping rule")
			continue
		}

		// Extract the single apiGroup (empty string "" for core API)
		apiGroup, ok := apiGroups[0].(string)
		if !ok {
			klog.Warning("ApiGroup is not a valid string, skipping rule")
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
		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := fieldMap["name"].(string)
			jsonPath, _ := fieldMap["jsonPath"].(string)
			dataTypeStr, dataTypeOK := fieldMap["type"].(string)
			//priority, _ := fieldMap["priority"].(string) // FUTURE: use this for additionalPrinterColumns extensions

			if name == "" || jsonPath == "" {
				klog.Warningf("Field missing name or jsonPath for resource %s, skipping", resourceKey)
				continue
			}

			/* TODO: come up with prefix schema before implementation. e.g. The specific configurable collection resource we read from determines the prefix to use
			user defined: user_myResource
			grc  defined: grc_thisPolicyThing
			virt defined: virt_thatVMWhatchamacallit
			*/
			name = "user_" + name

			extractProp := ExtractProperty{
				Name:     name,
				JSONPath: jsonPath,
				DataType: DataTypeString, // Default to string, matching CRD default
			}

			if dataTypeOK {
				extractProp.DataType = stringToDataType(dataTypeStr)
			}

			// FUTURE: collision handling with ExtractProperties that already exist in config, the first ExtractProperty
			// that gets parsed and stored in the node properties wins, the duplicate gets skipped
			resourceConfig.properties = append(resourceConfig.properties, extractProp)
			klog.V(2).Infof("Added custom field %s to resource %s", name, resourceKey)
		}

		// Update the merged config (not defaultTransformConfig)
		mergedTransformConfig[resourceKey] = resourceConfig
		klog.Infof("Merged %d custom fields for resource %s", len(fields), resourceKey)
	}

	klog.Info("Successfully merged configurable collection config")
}

// Helper function to safely access nested fields in unstructured data
func unstructuredNested(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj
	for _, field := range fields {
		m, ok := val.(map[string]interface{})
		if !ok {
			return nil, false, nil
		}
		val, ok = m[field]
		if !ok {
			return nil, false, nil
		}
	}
	return val, true, nil
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
