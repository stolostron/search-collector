package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.ConfigMap and yields a Node
func transformConfigMap(resource *v1.ConfigMap) Node {

	configMap := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	configMap.Properties["kind"] = "ConfigMap"

	return configMap
}
