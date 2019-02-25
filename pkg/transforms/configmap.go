package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.ConfigMap and yields a rg.Node
func TransformConfigMap(resource *v1.ConfigMap) rg.Node {

	props := CommonProperties(resource) // Start off with the common properties

	// Form these properties into an rg.Node
	return rg.Node{
		Label:      "ConfigMap",
		Properties: props,
	}
}
