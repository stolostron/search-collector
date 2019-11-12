/*
IBM Confidential
OCO Source Materials
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
	node := transformCommon(r)         // Start off with the common properties
	apiGroupVersion(r.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["current"] = int64(r.Status.Replicas)
	node.Properties["desired"] = int64(0)
	if r.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*r.Spec.Replicas)
	}

	return node
}

func (r ReplicaSetResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(r.ReplicaSet.UID)
	rsNode := ns.ByUID[UID]
	nodeInfo := NodeInfo{Name: r.Name, NameSpace: r.Namespace, UID: UID, EdgeType: "ownedBy", Kind: r.Kind}

	//ownedBy edges
	if rsNode.GetMetadata("OwnerUID") != "" {
		ret = append(ret, edgesByOwner(rsNode.GetMetadata("OwnerUID"), ns, nodeInfo)...)
	}

	return ret
}
