package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Namespace and yields a Node
func TransformNamespace(resource *v1.Namespace) Node {

	namespace := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	namespace.Properties["kind"] = "Namespace"
	namespace.Properties["status"] = string(resource.Status.Phase)

	return namespace
}
