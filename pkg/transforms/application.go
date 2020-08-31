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

	app "sigs.k8s.io/application/api/v1beta1"
)

// ApplicationResource ...
type ApplicationResource struct {
	node        Node
	annotations map[string]string
}

// ApplicationResourceBuilder ...
func ApplicationResourceBuilder(a *app.Application) *ApplicationResource {
	node := transformCommon(a)
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["dashboard"] = a.GetAnnotations()["apps.open-cluster-management.io/dashboard"]

	return &ApplicationResource{node: node, annotations: a.GetAnnotations()}
}

// BuildNode construct the node for the Application Resources
func (a ApplicationResource) BuildNode() Node {
	return a.node
}

// BuildEdges construct the edges for the Application Resources
// See documentation at pkg/transforms/README.md
func (a ApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := a.node.UID

	nodeInfo := NodeInfo{
		NameSpace: a.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "contains",
		Kind:      a.node.Properties["kind"].(string),
		Name:      a.node.Properties["name"].(string)}

	if len(a.annotations["apps.open-cluster-management.io/deployables"]) > 0 {
		deployableMap := make(map[string]struct{})
		for _, deployable := range strings.Split(a.annotations["apps.open-cluster-management.io/deployables"], ",") {
			deployableMap[deployable] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns)...)
	}

	if len(a.annotations["apps.open-cluster-management.io/subscriptions"]) > 0 {
		subscriptionMap := make(map[string]struct{})
		for _, subscription := range strings.Split(a.annotations["apps.open-cluster-management.io/subscriptions"], ",") {
			subscriptionMap[subscription] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(subscriptionMap, "Subscription", nodeInfo, ns)...)
	}

	if len(a.annotations["apps.open-cluster-management.io/placementbindings"]) > 0 {
		placementBindingMap := make(map[string]struct{})
		for _, pBinding := range strings.Split(a.annotations["apps.open-cluster-management.io/placementbindings"], ",") {
			placementBindingMap[pBinding] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(placementBindingMap, "PlacementBinding", nodeInfo, ns)...)
	}
	return ret
}
