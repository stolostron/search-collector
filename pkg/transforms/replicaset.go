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

// Takes a *v1.ReplicaSet and yields a Node
func transformReplicaSet(resource *v1.ReplicaSet) Node {

	replicaSet := transformCommon(resource) // Start off with the common properties

	// Find the resource's owner. Resources can have multiple ownerReferences, but only one controller.
	ownerUID := ""
	for _, ref := range resource.OwnerReferences {
		if *ref.Controller {
			ownerUID = prefixedUID(ref.UID) // TODO prefix with clustername
			continue
		}
	}

	// Extract the properties specific to this type
	replicaSet.Properties["kind"] = "ReplicaSet"
	replicaSet.Properties["apigroup"] = "apps"
	replicaSet.Properties["current"] = int64(resource.Status.Replicas)
	replicaSet.Properties["desired"] = int64(0)
	replicaSet.Properties["_ownerUID"] = ownerUID
	if resource.Spec.Replicas != nil {
		replicaSet.Properties["desired"] = int64(*resource.Spec.Replicas)
	}

	return replicaSet
}
