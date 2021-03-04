// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"
)

func TestTransformPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("policyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	// Test unique fields that exist in policy report and are shown in UI - the common test will test the other bits
	AssertEqual("message", node.Properties["message"], string("policyreport testing risk 1"), t)
	AssertDeepEqual("category", node.Properties["category"], []string{"category", "category1", "category2"}, t)
	AssertEqual("risk", node.Properties["risk"], string("1"), t)
}

func TestPolicyReportBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource policyreport.json
	var pr PolicyReport
	UnmarshalFile("policyreport.json", &pr, t)
	edges := PolicyReportResourceBuilder(&pr).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("PolicyReport has no edges:", len(edges), 0, t)
}
