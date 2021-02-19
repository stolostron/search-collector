/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"testing"

	v1 "k8s.io/api/batch/v1"
)

func TestTransformJob(t *testing.T) {
	var j v1.Job
	UnmarshalFile("job.json", &j, t)
	node := JobResourceBuilder(&j).BuildNode()

	// Test only the fields that exist in job - the common test will test the other bits
	AssertEqual("successful", node.Properties["successful"], int64(1), t)
	AssertEqual("completions", node.Properties["completions"], int64(1), t)
	AssertEqual("parallelism", node.Properties["parallelism"], int64(1), t)
}

func TestJobBuildEdges(t *testing.T) {
	var j v1.Job
	UnmarshalFile("job.json", &j, t)

	store := NodeStore{
		ByUID:               make(map[string]Node),
		ByKindNamespaceName: make(map[string]map[string]map[string]Node),
	}

	edges := JobResourceBuilder(&j).BuildEdges(store)

	AssertEqual("Job has no edges:", len(edges), 0, t)
}
