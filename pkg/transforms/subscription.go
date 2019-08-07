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

	app "github.ibm.com/IBMMulticloudPlatform/subscription/pkg/apis/app/v1alpha1"
)

type SubscriptionResource struct {
	*app.Subscription
}

func (s SubscriptionResource) BuildNode() Node {
	node := transformCommon(s)

	node.Properties["kind"] = "Subscription"
	if s.Spec.Package != "" {
		node.Properties["package"] = string(s.Spec.Package)
	}
	if s.Spec.PackageFilter != nil && s.Spec.PackageFilter.Version != "" {
		node.Properties["packageFilterVersion"] = string(s.Spec.PackageFilter.Version)
	}
	if s.Spec.Channel != "" {
		node.Properties["channel"] = s.Spec.Channel
	}
	if s.GetAnnotations()["_app.ibm.com/hosting-subscription"] != "" {
		node.Properties["_hostingSubscription"] = s.GetAnnotations()["_app.ibm.com/hosting-subscription"]
	}
	//TODO: Add property Status
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
		ret = append(ret, edgesByDestinationName(channelMap, ret, "Channel", nodeInfo, ns)...)
	}

	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)

	return ret
}
