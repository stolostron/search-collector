package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

// Takes a *v1.ReplicaSet and yields a Node
func transformReplicaSet(resource *v1.ReplicaSet) Node {

	replicaSet := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	replicaSet.Properties["kind"] = "ReplicaSet"
	replicaSet.Properties["apigroup"] = "apps"
	replicaSet.Properties["current"] = int64(resource.Status.Replicas)
	replicaSet.Properties["desired"] = int64(0)
	if resource.Spec.Replicas != nil {
		replicaSet.Properties["desired"] = int64(*resource.Spec.Replicas)
	}

	return replicaSet
}
