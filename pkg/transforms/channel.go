/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	app "github.ibm.com/IBMMulticloudPlatform/channel/pkg/apis/app/v1alpha1"
)

type ChannelResource struct {
	*app.Channel
}

func (c ChannelResource) BuildNode() Node {
	node := transformCommon(c)

	node.Properties["kind"] = "Channel"
	node.Properties["type"] = string(c.Spec.Type)
	node.Properties["pathname"] = c.Spec.PathName

	return node
}

func (c ChannelResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(c.UID)

	nodeInfo := NodeInfo{NameSpace: c.Namespace, UID: UID, EdgeType: "uses", Kind: c.Kind, Name: c.Name}

	//Build uses edges to connect channel to configmaps and secrets
	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})
	if c.Spec.ConfigMapRef != nil {
		configmapMap[c.Spec.ConfigMapRef.Name] = struct{}{}
	}
	if c.Spec.SecretRef != nil {
		secretMap[c.Spec.SecretRef.Name] = struct{}{}
	}

	ret = append(ret, edgesByDestinationName(secretMap, ret, "Secret", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(configmapMap, ret, "ConfigMap", nodeInfo, ns)...)

	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
