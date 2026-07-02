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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

func TestTransformCronJob(t *testing.T) {
	var c v1.CronJob
	UnmarshalFile("cronjob.json", &c, t)
	node := CronJobResourceBuilder(&c, &unstructured.Unstructured{}).BuildNode()

	// Build time struct matching time in test data
	date := time.Date(2019, 3, 5, 23, 30, 0, 0, time.UTC)

	// Test only the fields that exist in cronjob - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "CronJob", t)
	AssertEqual("active", node.Properties["active"], int64(0), t)
	AssertEqual("lastSchedule", node.Properties["lastSchedule"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("schedule", node.Properties["schedule"], "30 23 * * *", t)
	AssertEqual("suspend", node.Properties["suspend"], "false", t)
}

// TestBooleanFieldsStoredAsStrings_CronJob verifies that the suspend field on
// CronJob is stored as a string so the search API can query it correctly.
func TestBooleanFieldsStoredAsStrings_CronJob(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		suspend  *bool
		expected string
	}{
		{"nil pointer defaults to false", nil, "false"},
		{"false pointer", &falseVal, "false"},
		{"true pointer", &trueVal, "true"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := v1.CronJob{}
			c.Spec.Suspend = tc.suspend
			node := CronJobResourceBuilder(&c, &unstructured.Unstructured{}).BuildNode()
			val, ok := node.Properties["suspend"].(string)
			if !ok {
				t.Fatalf("suspend should be a string, got %T: %v", node.Properties["suspend"], node.Properties["suspend"])
			}
			AssertEqual("suspend", val, tc.expected, t)
		})
	}
}

func TestCronJobBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource cronjob.json
	var cron v1.CronJob
	UnmarshalFile("cronjob.json", &cron, t)
	edges := CronJobResourceBuilder(&cron, &unstructured.Unstructured{}).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("CronJob has no edges:", len(edges), 0, t)
}
