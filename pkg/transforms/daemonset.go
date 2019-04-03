package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.DaemonSet and yields a Node
func transformDaemonSet(resource *v1.DaemonSet) Node {

	daemonSet := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	daemonSet.Properties["kind"] = "DaemonSet"
	daemonSet.Properties["available"] = int64(resource.Status.NumberAvailable)
	daemonSet.Properties["current"] = int64(resource.Status.CurrentNumberScheduled)
	daemonSet.Properties["desired"] = int64(resource.Status.DesiredNumberScheduled)
	daemonSet.Properties["ready"] = int64(resource.Status.NumberReady)
	daemonSet.Properties["updated"] = int64(resource.Status.UpdatedNumberScheduled)

	return daemonSet
}
