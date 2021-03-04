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

	v1 "k8s.io/api/apps/v1"
)

func TestTransformDaemonSet(t *testing.T) {
	var d v1.DaemonSet
	UnmarshalFile("daemonset.json", &d, t)
	node := DaemonSetResourceBuilder(&d).BuildNode()

	// Test only the fields that exist in daemonset - the common test will test the other bits
	AssertEqual("available", node.Properties["available"], int64(1), t)
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
	AssertEqual("ready", node.Properties["ready"], int64(1), t)
	AssertEqual("updated", node.Properties["updated"], int64(1), t)
}

func TestDaemonSetBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource daemonset.json
	var ds v1.DaemonSet
	UnmarshalFile("daemonset.json", &ds, t)
	edges := DaemonSetResourceBuilder(&ds).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("DaemonSet has no edges:", len(edges), 0, t)
}
