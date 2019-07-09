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

type ReplicaSetResource struct {
	*v1.ReplicaSet
}

func (r ReplicaSetResource) BuildNode() Node {
	node := transformCommon(r) // Start off with the common properties

	// Find the resource's owner. Resources can have multiple ownerReferences, but only one controller.
	ownerUID := ""
	for _, ref := range r.OwnerReferences {
		if *ref.Controller {
			ownerUID = prefixedUID(ref.UID) // TODO prefix with clustername
			continue
		}
	}

	// Extract the properties specific to this type
	node.Properties["kind"] = "ReplicaSet"
	node.Properties["apigroup"] = "apps"
	node.Properties["current"] = int64(r.Status.Replicas)
	node.Properties["desired"] = int64(0)
	node.Properties["_ownerUID"] = ownerUID
	if r.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*r.Spec.Replicas)
	}

	return node
}

func (r ReplicaSetResource) BuildEdges(state map[string]Node) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
