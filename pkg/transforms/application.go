/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strings"

	v1 "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
)

type ApplicationResource struct {
	*v1.Application
}

func (a ApplicationResource) BuildNode() Node {
	node := transformCommon(a)
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["dashboard"] = a.GetAnnotations()["app.open-cluster-management.io/dashboard"]

	return node
}

func (a ApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(a.UID)

	nodeInfo := NodeInfo{NameSpace: a.Namespace, UID: UID, EdgeType: "contains", Kind: a.Kind, Name: a.Name}

	if len(a.GetAnnotations()["app.open-cluster-management.io/deployables"]) > 0 {
		deployableMap := make(map[string]struct{})
		for _, deployable := range strings.Split(a.GetAnnotations()["app.open-cluster-management.io/deployables"], ",") {
			deployableMap[deployable] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(deployableMap, "Deployable", nodeInfo, ns)...)
	}

	if len(a.GetAnnotations()["app.open-cluster-management.io/subscriptions"]) > 0 {
		subscriptionMap := make(map[string]struct{})
		for _, subscription := range strings.Split(a.GetAnnotations()["app.open-cluster-management.io/subscriptions"], ",") {
			subscriptionMap[subscription] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(subscriptionMap, "Subscription", nodeInfo, ns)...)
	}

	if len(a.GetAnnotations()["app.open-cluster-management.io/placementbindings"]) > 0 {
		placementBindingMap := make(map[string]struct{})
		for _, placementBinding := range strings.Split(a.GetAnnotations()["app.open-cluster-management.io/placementbindings"], ",") {
			placementBindingMap[placementBinding] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(placementBindingMap, "PlacementBinding", nodeInfo, ns)...)
	}
	return ret
}
