/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strings"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

type EventResource struct {
	*v1.Event
}

func (e EventResource) BuildNode() Node {
	node := transformCommon(e)

	// Extract the properties specific to this type
	node.Properties["kind"] = "Event"
	node.Properties["source"] = string(e.Source.Component)
	node.Properties["InvolvedObject.kind"] = e.InvolvedObject.Kind
	node.Properties["InvolvedObject.name"] = e.InvolvedObject.Name
	node.Properties["InvolvedObject.namespace"] = e.InvolvedObject.Namespace
	node.Properties["InvolvedObject.uid"] = string(e.InvolvedObject.UID)
	node.Properties["message"] = e.Message
	node.Properties["type"] = e.Type
	// We need to get the resource ID and Resource name from the message .
	// Sample format may look like this --> "Detected pod(uid:Uid) container default/ubi7-2-pod/ubi7-2-pod"
	// WE should not get any other kinds , but just another check
	if e.InvolvedObject.Kind == "VulnerabilityPolicy" || e.InvolvedObject.Kind == "MutationPolicy" {
		//get UID value
		if strings.Contains(e.Message, "uid:") {
			parts := strings.Split(e.Message, "uid:")
			uidstr := strings.Split(parts[1], ")")
			node.Properties["message.uid"] = strings.TrimSpace(uidstr[0])
		} else {
			glog.Warningf("Event ID: %s has message with no uid ", string(e.UID))
		}
		// Get namespace,resource,container names
		strFields := strings.Fields(e.Message)
		fieldCount := len(strFields)
		resourceStr := strFields[fieldCount-1]
		if strings.Contains(resourceStr, "/") {
			parts := strings.Split(resourceStr, "/")
			if len(parts) == 2 {
				node.Properties["message.namespace"] = parts[0]
				node.Properties["message.resourceName"] = parts[1]
			}
			if len(parts) == 3 {
				node.Properties["message.namespace"] = parts[0]
				node.Properties["message.resourceName"] = parts[1]
				node.Properties["message.containerName"] = parts[2]

			}
		} else {
			glog.Warningf("Event ID: %s has message with no resource name ", string(e.UID))
		}

	}

	return node
}

func (v EventResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}
