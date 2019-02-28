package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.StatefulSet and yields a Node
func TransformStatefulSet(resource *v1.StatefulSet) Node {

	statefulSet := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	statefulSet.Properties["kind"] = "StatefulSet"
	statefulSet.Properties["current"] = resource.Status.Replicas
	statefulSet.Properties["desired"] = resource.Spec.Replicas

	return statefulSet
}
