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
