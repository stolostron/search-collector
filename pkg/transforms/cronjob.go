/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

// CronJobResource ...
type CronJobResource struct {
	node Node
}

// CronJobResourceBuilder ...
func CronJobResourceBuilder(c *v1.CronJob) *CronJobResource {
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

	return &CronJobResource{node: node}
}

// BuildNode construct the node for the Cronjob Resources
func (c CronJobResource) BuildNode() Node {
	return c.node
}

// BuildEdges construct the edges for the Cronjob Resources
func (c CronJobResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
