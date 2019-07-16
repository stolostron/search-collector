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

type ApplicationRelationshipResource struct {
	*mcm.ApplicationRelationship
}

func (a ApplicationRelationshipResource) BuildNode() Node {
	node := transformCommon(a)

	node.Properties["kind"] = "ApplicationRelationship"
	node.Properties["apigroup"] = "mcm.ibm.com"
	node.Properties["destination"] = a.Spec.Destination.Name
	node.Properties["source"] = a.Spec.Source.Name
	node.Properties["type"] = string(a.Spec.RelType)

	return node
}

func (a ApplicationRelationshipResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
