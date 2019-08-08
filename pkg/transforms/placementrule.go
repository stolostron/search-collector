/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	app "github.ibm.com/IBMMulticloudPlatform/placementrule/pkg/apis/app/v1alpha1"
)

type PlacementRuleResource struct {
	*app.PlacementRule
}

func (p PlacementRuleResource) BuildNode() Node {
	node := transformCommon(p) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "PlacementRule"
	node.Properties["apigroup"] = "app.ibm.com"
	//TODO: Add other properties
	return node
}

func (p PlacementRuleResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
