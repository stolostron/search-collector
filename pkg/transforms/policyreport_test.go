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
	AssertDeepEqual("category Length", len(node.Properties["category"].([]string)), 5, t)
	AssertDeepEqual("policyViolations", node.Properties["policyViolations"], []string{"policyreport testing risk 1 policy", "policyreport testing risk 2 policy"}, t)
	AssertDeepEqual("numPolicyViolations", node.Properties["numPolicyViolations"], 2, t)
	AssertDeepEqual("critical", node.Properties["critical"], 0, t)
	AssertDeepEqual("important", node.Properties["important"], 0, t)
	AssertDeepEqual("moderate", node.Properties["moderate"], 1, t)
	AssertDeepEqual("low", node.Properties["low"], 1, t)

	AssertDeepEqual("scope", node.Properties["scope"], "test-cluster", t)
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
