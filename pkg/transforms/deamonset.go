package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.DaemonSet and yields a Node
func TransformDaemonSet(resource *v1.DaemonSet) Node {

	daemonSet := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	daemonSet.Properties["kind"] = "DaemonSet"
	daemonSet.Properties["available"] = resource.Status.NumberAvailable
	daemonSet.Properties["current"] = resource.Status.CurrentNumberScheduled
	daemonSet.Properties["desired"] = resource.Status.DesiredNumberScheduled
	daemonSet.Properties["ready"] = resource.Status.NumberReady
	daemonSet.Properties["updated"] = resource.Status.UpdatedNumberScheduled

	return daemonSet
}
