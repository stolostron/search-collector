/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	v1 "k8s.io/api/core/v1"
)

type NamespaceResource struct {
	*v1.Namespace
}

func (n NamespaceResource) BuildNode() Node {
	node := transformCommon(n)         // Start off with the common properties
	apiGroupVersion(n.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["status"] = string(n.Status.Phase)

	return node
}

func (n NamespaceResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
