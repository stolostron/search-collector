/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	mcm "github.com/IBM/multicloud-operators-deployable/pkg/apis/app/v1alpha1"
)

type ApplicationRelationshipResource struct {
	*mcm.ApplicationRelationship
}

func (a ApplicationRelationshipResource) BuildNode() Node {
	node := transformCommon(a)

	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["destination"] = a.Spec.Destination.Name
	node.Properties["source"] = a.Spec.Source.Name
	node.Properties["type"] = string(a.Spec.RelType)

	return node
}

func (a ApplicationRelationshipResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
