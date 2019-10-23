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

type StatefulSetResource struct {
	*v1.StatefulSet
}

func (s StatefulSetResource) BuildNode() Node {
	node := transformCommon(s)         // Start off with the common properties
	apiGroupVersion(s.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["current"] = int64(s.Status.Replicas)
	node.Properties["desired"] = int64(0)
	if s.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*s.Spec.Replicas)
	}

	return node
}

func (s StatefulSetResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(s.UID)
	//deployer subscriber edges
	nodeInfo := NodeInfo{NameSpace: s.Namespace, UID: UID, Kind: s.Kind, Name: s.Name}
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
