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
	node := transformCommon(s) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "StatefulSet"
	node.Properties["apigroup"] = "apps"
	node.Properties["current"] = int64(s.Status.Replicas)
	node.Properties["desired"] = int64(0)
	if s.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*s.Spec.Replicas)
	}

	return node
}

func (s StatefulSetResource) BuildEdges(state map[string]Node) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
