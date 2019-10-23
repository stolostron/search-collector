/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

type CronJobResource struct {
	*v1.CronJob
}

func (c CronJobResource) BuildNode() Node {
	node := transformCommon(c) // Start off with the common properties

	apiGroupVersion(c.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["active"] = int64(len(c.Status.Active))
	node.Properties["schedule"] = c.Spec.Schedule
	node.Properties["lastSchedule"] = ""
	if c.Status.LastScheduleTime != nil {
		node.Properties["lastSchedule"] = c.Status.LastScheduleTime.UTC().Format(time.RFC3339)
	}
	node.Properties["suspend"] = false
	if c.Spec.Suspend != nil {
		node.Properties["suspend"] = *c.Spec.Suspend
	}

	return node
}

func (c CronJobResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(c.UID)
	//deployer subscriber edges
	nodeInfo := NodeInfo{NameSpace: c.Namespace, UID: UID, Kind: c.Kind, Name: c.Name}
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}
