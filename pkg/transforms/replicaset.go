/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"github.com/golang/glog"
	v1 "k8s.io/api/apps/v1"
)

type ReplicaSetResource struct {
	*v1.ReplicaSet
}

func (r ReplicaSetResource) BuildNode() Node {
	node := transformCommon(r) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "ReplicaSet"
	node.Properties["apigroup"] = "apps"
	node.Properties["current"] = int64(r.Status.Replicas)
	node.Properties["desired"] = int64(0)
	if r.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*r.Spec.Replicas)
	}

	return node
}

func (r ReplicaSetResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}

	//ownedBy edge
	ownerUID := ""
	UID := prefixedUID(r.ReplicaSet.UID)

	// Find the resource's owner. Resources can have multiple ownerReferences, but only one controller.
	for _, ref := range r.ReplicaSet.OwnerReferences {

		if *ref.Controller {
			ownerUID = prefixedUID(ref.UID) // TODO prefix with clustername
			continue
		}
	}
	//Check if node referred to by ownerUID exists
	if _, ok := ns.ByUID[ownerUID]; ok {
		ret = append(ret, Edge{
			SourceUID: UID,
			DestUID:   ownerUID,
			EdgeType:  "ownedBy",
		})
	} else {
		glog.Infof("ownedBy edge not created as node with ownerUID %s doesn't exist.", ownerUID)
	}

	return ret
}
