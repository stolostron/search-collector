/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

func TestTransformCronJob(t *testing.T) {
	var c v1.CronJob
	UnmarshalFile("cronjob.json", &c, t)
	node := CronJobResourceBuilder(&c).BuildNode()

	// Build time struct matching time in test data
	date := time.Date(2019, 3, 5, 23, 30, 0, 0, time.UTC)

	// Test only the fields that exist in cronjob - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "CronJob", t)
	AssertEqual("active", node.Properties["active"], int64(0), t)
	AssertEqual("lastSchedule", node.Properties["lastSchedule"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("schedule", node.Properties["schedule"], "30 23 * * *", t)
	AssertEqual("suspend", node.Properties["suspend"], false, t)
}
