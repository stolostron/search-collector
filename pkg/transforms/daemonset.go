/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.DaemonSet and yields a Node
func transformDaemonSet(resource *v1.DaemonSet) Node {

	daemonSet := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	daemonSet.Properties["kind"] = "DaemonSet"
	daemonSet.Properties["apigroup"] = "apps"
	daemonSet.Properties["available"] = int64(resource.Status.NumberAvailable)
	daemonSet.Properties["current"] = int64(resource.Status.CurrentNumberScheduled)
	daemonSet.Properties["desired"] = int64(resource.Status.DesiredNumberScheduled)
	daemonSet.Properties["ready"] = int64(resource.Status.NumberReady)
	daemonSet.Properties["updated"] = int64(resource.Status.UpdatedNumberScheduled)

	return daemonSet
}
