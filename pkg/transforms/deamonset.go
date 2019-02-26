package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.DaemonSet and yields a rg.Node
func TransformDaemonSet(resource *v1.DaemonSet) rg.Node {

	props := CommonProperties(resource) // Start off with the common properties

	// Extract the properties specific to this type
	props["available"] = resource.Status.NumberAvailable
	props["current"] = resource.Status.CurrentNumberScheduled
	props["desired"] = resource.Status.DesiredNumberScheduled
	props["ready"] = resource.Status.NumberReady
	props["updated"] = resource.Status.UpdatedNumberScheduled

	// Form these properties into an rg.Node
	return rg.Node{
		Label:      "DaemonSet",
		Properties: props,
	}
}
