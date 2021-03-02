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

	v1 "k8s.io/api/apps/v1"
)

func TestTransformStatefulSet(t *testing.T) {
	var s v1.StatefulSet
	UnmarshalFile("statefulset.json", &s, t)
	node := StatefulSetResourceBuilder(&s).BuildNode()

	// Test only the fields that exist in stateful set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
}

func TestStatefulSetBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource statefulset.json
	var ss v1.StatefulSet
	UnmarshalFile("statefulset.json", &ss, t)
	edges := StatefulSetResourceBuilder(&ss).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("StatefulSet has no edges:", len(edges), 0, t)
}
