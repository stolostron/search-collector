package transforms

import (
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Extracts the common properties from a default k8s resource of unknown type and returns them in a map ready to be put in an Node
func CommonProperties(resource machineryV1.Object) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["resourceVersion"] = resource.GetResourceVersion()
	ret["cluster"] = resource.GetClusterName()
	ret["name"] = resource.GetName()
	ret["namespace"] = resource.GetNamespace()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().String()
	ret["labels"] = resource.GetLabels()
	return ret
}

// Transforms a resource of unknown type by simply pulling out the common properties.
func TransformCommon(resource machineryV1.Object) Node {
	return Node{
		UID:        string(resource.GetUID()),
		Properties: CommonProperties(resource),
	}
}

// Extracts the properties from a non-default k8s resource and returns them in a map ready to be put in an Node
func UnstructuredProperties(resource *unstructured.Unstructured) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["kind"] = resource.GetKind()
	ret["resourceVersion"] = resource.GetResourceVersion()
	ret["cluster"] = resource.GetClusterName()
	ret["name"] = resource.GetName()
	ret["namespace"] = resource.GetNamespace()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().String()
	ret["labels"] = resource.GetLabels()
	return ret

}

// Transforms an unstrucuted.Unstructured (which represents a non-default k8s object) into a Node
func TransformUnstructured(resource *unstructured.Unstructured) Node {
	return Node{
		UID:        string(resource.GetUID()),
		Properties: UnstructuredProperties(resource),
	}
}
