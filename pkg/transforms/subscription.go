/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"strings"

	app "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
)

// SubscriptionResource ...
type SubscriptionResource struct {
	node        Node
	annotations map[string]string
	Spec        app.SubscriptionSpec
}

// SubscriptionResourceBuilder ...
func SubscriptionResourceBuilder(s *app.Subscription) *SubscriptionResource {
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
	// Add timeWindow property
	node.Properties["timeWindow"] = ""
	if s.Spec.TimeWindow != nil && s.Spec.TimeWindow.WindowType != "" {
		node.Properties["timeWindow"] = s.Spec.TimeWindow.WindowType
	}
	// Add localPlacement property
	node.Properties["localPlacement"] = nil
	if s.Spec.Placement != nil && s.Spec.Placement.Local != nil {
		node.Properties["localPlacement"] = *s.Spec.Placement.Local
	}
	// Add hidden properties for app annotations
	const appAnnotationPrefix string = "apps.open-cluster-management.io/"
	const gitType = "git"
	const oldGitType = "github"
	annotations := s.GetAnnotations()
	for _, annotation := range []string{"branch", "path", "commit"} {
		annotationValue := annotations[appAnnotationPrefix+gitType+"-"+annotation]
		if annotationValue == "" {
			// Try old version of the annotation - to be removed if/when these annotations are no longer supported
			annotationValue = annotations[appAnnotationPrefix+oldGitType+"-"+annotation]
		}
		if annotationValue != "" {
			node.Properties["_"+gitType+strings.ReplaceAll(annotation, "-", "")] = annotationValue
		}
	}
	// Add metadata specific to this type
	if len(s.Spec.Channel) > 0 {
		node.Metadata["_channels"] = s.Spec.Channel
	}

	// Need to pass annotations & spec so we can access them when building the edges.
	return &SubscriptionResource{node: node, annotations: s.GetAnnotations(), Spec: s.Spec}
}

// BuildNode construct the node for the Subscription Resources
func (s SubscriptionResource) BuildNode() Node {
	return s.node
}

// BuildEdges construct the edges for the Subscription Resources
// See documentation at pkg/transforms/README.md
func (s SubscriptionResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := s.node.UID

	nodeInfo := NodeInfo{
		NameSpace: s.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "to",
		Kind:      s.node.Properties["kind"].(string),
		Name:      s.node.Properties["name"].(string)}
	channelMap := make(map[string]struct{})

	// TODO: This will work only for subscription in hub cluster - confirm logic
	// TODO: Connect subscription and channel in remote cluster as they might not be in the same namespace
	if len(s.Spec.Channel) > 0 {
		for _, channel := range strings.Split(s.Spec.Channel, ",") {
			channelMap[channel] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(channelMap, "Channel", nodeInfo, ns, []string{})...)
	}
	// refersTo edges
	// Builds edges between subscription and placement rules
	if s.Spec.Placement != nil && s.Spec.Placement.PlacementRef != nil && s.Spec.Placement.PlacementRef.Name != "" {
		nodeInfo.EdgeType = "refersTo"
		placementRuleMap := make(map[string]struct{})
		placementRuleMap[s.Spec.Placement.PlacementRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementRuleMap, "PlacementRule", nodeInfo, ns, []string{})...)
	}
	//subscribesTo edges
	if len(s.annotations["apps.open-cluster-management.io/deployables"]) > 0 {
		nodeInfo.EdgeType = "subscribesTo"
		deployableMap := make(map[string]struct{})
		for _, deployable := range strings.Split(s.annotations["apps.open-cluster-management.io/deployables"], ",") {
			deployableMap[deployable] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns, []string{})...)
	}

	return ret
}
