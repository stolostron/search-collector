package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.Deployment and yields a rg.Node
func TransformDeployment(resource *v1.Deployment) rg.Node {

	props := CommonProperties(resource) // Start off with the common properties

	// Extract the properties specific to this type
	props["available"] = resource.Status.AvailableReplicas
	props["current"] = resource.Status.Replicas
	props["desired"] = resource.Spec.Replicas
	props["ready"] = resource.Status.ReadyReplicas

	// Form these properties into an rg.Node
	return rg.Node{
		Label:      "Deployment",
		Properties: props,
	}
}
