/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

type PlacementPolicyResource struct {
	*mcm.PlacementPolicy
}

func (p PlacementPolicyResource) BuildNode() Node {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["replicas"] = int64(0)
	if p.Spec.ClusterReplicas != nil {
		node.Properties["replicas"] = int64(*p.Spec.ClusterReplicas)
	}

	l := len(p.Status.Decisions)
	decisions := make([]string, l)
	for i := 0; i < l; i++ {
		decisions[i] = p.Status.Decisions[i].ClusterName
	}
	node.Properties["decisions"] = decisions

	return node
}

func (p PlacementPolicyResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
