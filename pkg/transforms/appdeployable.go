/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	appDeployable "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

type AppDeployableResource struct {
	*appDeployable.Deployable
}

func (d AppDeployableResource) BuildNode() Node {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	//TODO: Add properties, TEMPLATE-KIND   TEMPLATE-APIVERSION    AGE   STATUS
	if d.Status.Phase != "" {
		node.Properties["status"] = d.Status.Phase
	}
	return node
}

func (d AppDeployableResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(d.UID)

	nodeInfo := NodeInfo{NameSpace: d.Namespace, UID: UID, EdgeType: "promotedTo", Kind: d.Kind, Name: d.Name}
	channelMap := make(map[string]struct{})
	//promotedTo edges
	if d.Spec.Channels != nil {
		for _, channel := range d.Spec.Channels {
			channelMap[channel] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(channelMap, "Channel", nodeInfo, ns)...)
	}

	//refersTo edges
	//Builds edges between deployable and placement rule
	if d.Spec.Placement != nil && d.Spec.Placement.PlacementRef != nil && d.Spec.Placement.PlacementRef.Name != "" {
		nodeInfo.EdgeType = "refersTo"
		placementRuleMap := make(map[string]struct{})
		placementRuleMap[d.Spec.Placement.PlacementRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementRuleMap, "PlacementRule", nodeInfo, ns)...)
	}
	return ret
}
