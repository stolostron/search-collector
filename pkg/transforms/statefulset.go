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

// Takes a *v1.StatefulSet and yields a Node
func transformStatefulSet(resource *v1.StatefulSet) Node {

	statefulSet := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	statefulSet.Properties["kind"] = "StatefulSet"
	statefulSet.Properties["apigroup"] = "apps"
	statefulSet.Properties["current"] = int64(resource.Status.Replicas)
	statefulSet.Properties["desired"] = int64(0)
	if resource.Spec.Replicas != nil {
		statefulSet.Properties["desired"] = int64(*resource.Spec.Replicas)
	}

	return statefulSet
}
