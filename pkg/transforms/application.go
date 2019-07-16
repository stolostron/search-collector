/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
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
	//no op for now to implement interface
	return []Edge{}
}
