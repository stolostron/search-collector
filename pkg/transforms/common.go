package transforms

import "k8s.io/apimachinery/pkg/apis/meta/v1"

// Extracts the common properties from a given k8s resource and returns them in a map ready to be put in an rg.Node
func CommonProperties(resource v1.Object) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["uid"] = string(resource.GetUID())
	ret["resourceVersion"] = resource.GetResourceVersion()
	ret["cluster"] = resource.GetClusterName()
	ret["name"] = resource.GetName()
	ret["namespace"] = resource.GetNamespace()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().String()
	ret["labels"] = resource.GetLabels()

	return ret
}
