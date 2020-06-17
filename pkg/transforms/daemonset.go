/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	v1 "k8s.io/api/apps/v1"
)

type DaemonSetResource struct {
	*v1.DaemonSet
}

func (d DaemonSetResource) BuildNode() Node {
	node := transformCommon(d)         // Start off with the common properties
	apiGroupVersion(d.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["available"] = int64(d.Status.NumberAvailable)
	node.Properties["current"] = int64(d.Status.CurrentNumberScheduled)
	node.Properties["desired"] = int64(d.Status.DesiredNumberScheduled)
	node.Properties["ready"] = int64(d.Status.NumberReady)
	node.Properties["updated"] = int64(d.Status.UpdatedNumberScheduled)

	return node
}

func (d DaemonSetResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
