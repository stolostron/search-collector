/*
IBM Confidential
OCO Source Materials
5737-E67
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

	// Extract the properties specific to this type
	node.Properties["kind"] = "Application"
	node.Properties["apigroup"] = "app.k8s.io"
	node.Properties["dashboard"] = a.GetAnnotations()["apps.ibm.com/dashboard"]

	return node
}

func (a ApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(a.UID)

	//glog.Info("Annotations: ", a.ObjectMeta.Annotations)
	nodeInfo := NodeInfo{NameSpace: a.Namespace, UID: UID, EdgeType: "contains", Kind: a.Kind, Name: a.Name}

	if len(a.GetAnnotations()["apps.ibm.com/deployables"]) > 0 {
		deployableMap := make(map[string]struct{})
		for _, deployable := range strings.Split(a.GetAnnotations()["apps.ibm.com/deployables"], ",") {
			deployableMap[deployable] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(deployableMap, ret, "Deployable", nodeInfo, ns)...)
	}

	if len(a.GetAnnotations()["app.ibm.com/subscriptions"]) > 0 {
		subscriptionMap := make(map[string]struct{})
		for _, subscription := range strings.Split(a.GetAnnotations()["app.ibm.com/subscriptions"], ",") {
			subscriptionMap[subscription] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(subscriptionMap, ret, "Subscription", nodeInfo, ns)...)
	}

	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)

	return ret
}
