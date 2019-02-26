package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Extracts the common properties from a given k8s resource and returns them in a map ready to be put in an rg.Node
func CommonProperties(resource machineryV1.Object) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uid"] = string(resource.GetUID())
	ret["resourceVersion"] = resource.GetResourceVersion()
	ret["cluster"] = resource.GetClusterName()
	ret["name"] = resource.GetName()
	ret["namespace"] = resource.GetNamespace()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().String()

	// RedisGraph doesn't do nested properties. Flatten the map.
	flatLabels := flattenStringMap("label__", resource.GetLabels())

	// Put all the properties from flattened map into ret.
	for k, v := range flatLabels {
		ret[k] = v
	}
	return ret
}

// Transforms a resource of unknown type by simply pulling out the common properties.
func TransformCommon(resource machineryV1.Object) rg.Node {
	return rg.Node{
		Label:      "UNKNOWN", // TODO there should be a way to figure this out - unsure.
		Properties: CommonProperties(resource),
	}
}
