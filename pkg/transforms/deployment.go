package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.Deployment and yields a Node
func TransformDeployment(resource *v1.Deployment) Node {

	deployment := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	deployment.Properties["kind"] = "Deployment"
	deployment.Properties["available"] = resource.Status.AvailableReplicas
	deployment.Properties["current"] = resource.Status.Replicas
	deployment.Properties["ready"] = resource.Status.ReadyReplicas
	deployment.Properties["desired"] = int32(0)
	if resource.Spec.Replicas != nil {
		deployment.Properties["desired"] = *(resource.Spec.Replicas)
	}

	return deployment
}
