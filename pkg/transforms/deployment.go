package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.Deployment and yields a Node
func transformDeployment(resource *v1.Deployment) Node {

	deployment := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	deployment.Properties["kind"] = "Deployment"
	deployment.Properties["apigroup"] = "apps"
	deployment.Properties["available"] = int64(resource.Status.AvailableReplicas)
	deployment.Properties["current"] = int64(resource.Status.Replicas)
	deployment.Properties["ready"] = int64(resource.Status.ReadyReplicas)
	deployment.Properties["desired"] = int64(0)
	if resource.Spec.Replicas != nil {
		deployment.Properties["desired"] = int64(*resource.Spec.Replicas)
	}

	return deployment
}
