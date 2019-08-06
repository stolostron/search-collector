/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

type DeploymentResource struct {
	*v1.Deployment
}

func (d DeploymentResource) BuildNode() Node {
	node := transformCommon(d) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "Deployment"
	node.Properties["apigroup"] = "apps"
	node.Properties["available"] = int64(d.Status.AvailableReplicas)
	node.Properties["current"] = int64(d.Status.Replicas)
	node.Properties["ready"] = int64(d.Status.ReadyReplicas)
	node.Properties["desired"] = int64(0)
	if d.Spec.Replicas != nil {
		node.Properties["desired"] = int64(*d.Spec.Replicas)
	}

	return node
}

func (d DeploymentResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(d.UID)
	//deployer subscriber edges
	nodeInfo := NodeInfo{NameSpace: d.Namespace, UID: UID, Kind: d.Kind, Name: d.Name}
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
