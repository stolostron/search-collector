/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import "strings"

type SubscriptionResource struct {
	*app.Subscription
}

func (s SubscriptionResource) BuildNode() Node {
	node := transformCommon(s)
	apiGroupVersion(s.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	if s.Spec.Package != "" {
		node.Properties["package"] = string(s.Spec.Package)
	}
	if s.Spec.PackageFilter != nil && s.Spec.PackageFilter.Version != "" {
		node.Properties["packageFilterVersion"] = string(s.Spec.PackageFilter.Version)
	}
	if s.Spec.Channel != "" {
		node.Properties["channel"] = s.Spec.Channel
	}
	// Phase is Propagated if Subscription is in hub or Subscribed if it is in endpoint
	if s.Status.Phase != "" {
		node.Properties["status"] = s.Status.Phase
	}
	if len(s.Spec.Channel) > 0 {
		node.Metadata["_channels"] = s.Spec.Channel
	}

	return node
}

func (s SubscriptionResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(s.UID)

	nodeInfo := NodeInfo{NameSpace: s.Namespace, UID: UID, EdgeType: "to", Kind: s.Kind, Name: s.Name}
	channelMap := make(map[string]struct{})
	// TODO: This will work only for subscription in hub cluster - confirm logic
	// TODO: Connect subscription and channel in remote cluster as they might not be in the same namespace
	if len(s.Spec.Channel) > 0 {
		for _, channel := range strings.Split(s.Spec.Channel, ",") {
			channelMap[channel] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(channelMap, "Channel", nodeInfo, ns)...)
	}
	//refersTo edges
	//Builds edges between subscription and placement rule
	if s.Spec.Placement != nil && s.Spec.Placement.PlacementRef != nil && s.Spec.Placement.PlacementRef.Name != "" {
		nodeInfo.EdgeType = "refersTo"
		placementRuleMap := make(map[string]struct{})
		placementRuleMap[s.Spec.Placement.PlacementRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementRuleMap, "PlacementRule", nodeInfo, ns)...)
	}
	//subscribesTo edges
	if len(s.GetAnnotations()["app.ibm.com/deployables"]) > 0 {
		nodeInfo.EdgeType = "subscribesTo"
		deployableMap := make(map[string]struct{})
		for _, deployable := range strings.Split(s.GetAnnotations()["app.ibm.com/deployables"], ",") {
			deployableMap[deployable] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns)...)
	}

	return ret
}
