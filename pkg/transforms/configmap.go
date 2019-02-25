package transforms

import v1 "k8s.io/api/core/v1"

// MCM Search representation of a pod to be put into graphDB
type ConfigMapNode struct {
	CommonNodeProperties
	Kind string `json: kind`
}

// Takes a *v1.ConfigMap and yields a transforms.ConfigMapsNode
func TransformConfigMap(resource *v1.ConfigMap) ConfigMapNode {

	return ConfigMapNode{
		CommonNodeProperties: TransformCommon(resource),
		Kind:                 "ConfigMap",
	}
}
