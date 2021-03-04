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
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource job.json
	var j v1.Job
	UnmarshalFile("job.json", &j, t)
	edges := JobResourceBuilder(&j).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Job has no edges:", len(edges), 0, t)
}
