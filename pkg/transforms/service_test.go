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

	v1 "k8s.io/api/core/v1"
)

func TestTransformService(t *testing.T) {
	var s v1.Service
	UnmarshalFile("service.json", &s, t)
	node := ServiceResourceBuilder(&s).BuildNode()

	AssertEqual("kind", node.Properties["kind"], "Service", t)
}

func TestServiceBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := []Node{{
		UID: "local-cluster/uuid-fake-pod",
		Properties: map[string]interface{}{"kind": "Pod", "namespace": "default", "name": "fake-pod",
			"label": map[string]string{"app": "test-fixture-selector", "release": "test-fixture-selector-release"}},
	}}
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource cronjob.json
	var svc v1.Service
	UnmarshalFile("service.json", &svc, t)
	edges := ServiceResourceBuilder(&svc).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Service has no edges:", len(edges), 1, t)

	AssertEqual("Service usedBy: ", edges[0].DestKind, "Pod", t)
}
