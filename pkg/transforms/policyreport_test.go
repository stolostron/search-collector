// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("policyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	// Test unique fields that exist in policy report and are shown in UI - the common test will test the other bits
	AssertDeepEqual("category Length", len(node.Properties["category"].([]string)), 5, t)
	AssertDeepEqual("rules", node.Properties["rules"], []string{"policyreport testing risk 1 policy", "policyreport testing risk 2 policy"}, t)
	AssertDeepEqual("numRuleViolations", node.Properties["numRuleViolations"], 2, t)
	AssertDeepEqual("critical", node.Properties["critical"], 0, t)
	AssertDeepEqual("important", node.Properties["important"], 0, t)
	AssertDeepEqual("moderate", node.Properties["moderate"], 1, t)
	AssertDeepEqual("low", node.Properties["low"], 1, t)

	AssertDeepEqual("scope", node.Properties["scope"], "test-cluster", t)
	expected := map[string]int{
		"policyreport testing risk 1 policy": 1,
		"policyreport testing risk 2 policy": 1,
	}
	AssertDeepEqual("_policyViolationCounts", node.Properties["_policyViolationCounts"].(map[string]int), expected, t)
}

func TestTransformKyvernoClusterPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("kyverno-clusterpolicyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	AssertDeepEqual("category", node.Properties["category"].([]string), []string{"Kubecost"}, t)
	AssertDeepEqual("rules", node.Properties["rules"], []string{"no-label-of-monkey", "require-kubecost-labels"}, t)
	// 1 failure and 1 error
	AssertDeepEqual("numRuleViolations", node.Properties["numRuleViolations"], 2, t)
	expected := map[string]int{"require-kubecost-labels": 2, "no-label-of-monkey": 0}
	AssertDeepEqual("_policyViolationCounts", node.Properties["_policyViolationCounts"].(map[string]int), expected, t)
}

func TestTransformKyvernoPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("kyverno-policyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	AssertDeepEqual("category", node.Properties["category"].([]string), []string{"Kubecost"}, t)
	AssertDeepEqual(
		"rules",
		node.Properties["rules"],
		[]string{"open-cluster-management-agent-addon/require-kubecost-labels", "require-kubecost-labels"},
		t,
	)
	AssertDeepEqual("numRuleViolations", node.Properties["numRuleViolations"], 2, t)
	expected := map[string]int{
		"require-kubecost-labels":                                     1,
		"open-cluster-management-agent-addon/require-kubecost-labels": 1,
	}
	AssertDeepEqual("_policyViolationCounts", node.Properties["_policyViolationCounts"].(map[string]int), expected, t)
}

func TestKyvernoClusterPolicyReportBuildEdges(t *testing.T) {
	p := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "require-kubecost-labels",
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "critical",
				},
				"uid": "132ec5b8-892b-40da-8b92-af141c377dfe",
			},
			"spec": map[string]interface{}{
				"validationFailureAction": "deny",
				"random":                  "value",
			},
		},
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "kube-public",
				"uid":  "13967788-e842-4662-9630-a9e9c39fa199",
			},
		},
	}

	nodes := []Node{KyvernoPolicyResourceBuilder(p).node, GenericResourceBuilder(ns).node}
	nodeStore := BuildFakeNodeStore(nodes)

	var pr PolicyReport
	UnmarshalFile("kyverno-clusterpolicyreport.json", &pr, t)

	edges := PolicyReportResourceBuilder(&pr).BuildEdges(nodeStore)

	if len(edges) != 3 {
		t.Fatalf("Expected two edges but got %d", len(edges))
	}

	edge := edges[0]
	expectedEdge := Edge{
		SourceUID:  "local-cluster/132ec5b8-892b-40da-8b92-af141c377dfe",
		SourceKind: "ClusterPolicy",
		EdgeType:   "reports",
		DestUID:    "local-cluster/509de4c9-ed73-4309-9764-c88334781eae",
		DestKind:   "ClusterPolicyReport",
	}
	AssertDeepEqual("edge", edge, expectedEdge, t)

	edge2 := edges[1]
	expectedEdge2 := Edge{
		SourceUID:  "local-cluster/132ec5b8-892b-40da-8b92-af141c377dfe",
		SourceKind: "ClusterPolicy",
		EdgeType:   "appliesTo",
		DestUID:    "local-cluster/13967788-e842-4662-9630-a9e9c39fa199",
		DestKind:   "Namespace",
	}
	AssertDeepEqual("edge", edge2, expectedEdge2, t)

	edge3 := edges[2]
	expectedEdge3 := Edge{
		SourceUID:  "local-cluster/509de4c9-ed73-4309-9764-c88334781eae",
		SourceKind: "ClusterPolicyReport",
		EdgeType:   "reportsOn",
		DestUID:    "local-cluster/13967788-e842-4662-9630-a9e9c39fa199",
		DestKind:   "Namespace",
	}
	AssertDeepEqual("edge", edge3, expectedEdge3, t)
}

func TestKyvernoPolicyReportBuildEdges(t *testing.T) {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "Policy",
			"metadata": map[string]interface{}{
				"name":      "require-kubecost-labels",
				"namespace": "open-cluster-management-agent-addon",
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "132ec5b8-892b-40da-8b92-af141c377dfe",
			},
			"spec": map[string]interface{}{
				"validationFailureAction": "deny",
				"random":                  "value",
			},
		},
	}

	clusterPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "require-kubecost-labels",
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "162ec5b8-892b-40da-8b92-af141c377ddd",
			},
			"spec": map[string]interface{}{
				"validationFailureAction": "deny",
				"random":                  "value",
			},
		},
	}

	violatingPod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "require-kubecost-labels",
				"namespace": "open-cluster-management-agent-addon",
				"uid":       "019db722-d5c7-4085-9778-d8ebc33e95f2",
			},
		},
	}

	nodes := []Node{
		KyvernoPolicyResourceBuilder(policy).node,
		KyvernoPolicyResourceBuilder(clusterPolicy).node,
		GenericResourceBuilder(violatingPod).node,
	}
	nodeStore := BuildFakeNodeStore(nodes)

	var pr PolicyReport
	UnmarshalFile("kyverno-policyreport.json", &pr, t)

	// Test a PolicyReport generated by a Policy kind
	edges := PolicyReportResourceBuilder(&pr).BuildEdges(nodeStore)

	if len(edges) != 6 {
		t.Fatalf("Expected 6 edges but got %d", len(edges))
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].SourceUID == edges[j].SourceUID {
			return edges[i].DestUID < edges[j].DestUID
		}

		return edges[i].SourceUID < edges[j].SourceUID
	})

	edge1 := edges[0]
	expectedEdge1 := Edge{
		SourceUID:  "local-cluster/132ec5b8-892b-40da-8b92-af141c377dfe",
		SourceKind: "Policy",
		EdgeType:   "appliesTo",
		DestUID:    "local-cluster/019db722-d5c7-4085-9778-d8ebc33e95f2",
		DestKind:   "Pod",
	}
	AssertDeepEqual("edge", edge1, expectedEdge1, t)

	edge2 := edges[1]
	expectedEdge2 := Edge{
		SourceUID:  "local-cluster/132ec5b8-892b-40da-8b92-af141c377dfe",
		SourceKind: "Policy",
		EdgeType:   "reports",
		DestUID:    "local-cluster/53cd0e2e-34e0-454b-a0c4-e4dbf9306470",
		DestKind:   "PolicyReport",
	}
	AssertDeepEqual("edge", edge2, expectedEdge2, t)

	edge3 := edges[2]
	expectedEdge3 := Edge{
		SourceUID:  "local-cluster/162ec5b8-892b-40da-8b92-af141c377ddd",
		SourceKind: "ClusterPolicy",
		EdgeType:   "appliesTo",
		DestUID:    "local-cluster/019db722-d5c7-4085-9778-d8ebc33e95f2",
		DestKind:   "Pod",
	}
	AssertDeepEqual("edge", edge3, expectedEdge3, t)

	edge4 := edges[3]
	expectedEdge4 := Edge{
		SourceUID:  "local-cluster/162ec5b8-892b-40da-8b92-af141c377ddd",
		SourceKind: "ClusterPolicy",
		EdgeType:   "reports",
		DestUID:    "local-cluster/53cd0e2e-34e0-454b-a0c4-e4dbf9306470",
		DestKind:   "PolicyReport",
	}
	AssertDeepEqual("edge", edge4, expectedEdge4, t)

	edge5 := edges[4]
	expectedEdge5 := Edge{
		SourceUID:  "local-cluster/53cd0e2e-34e0-454b-a0c4-e4dbf9306470",
		SourceKind: "PolicyReport",
		EdgeType:   "reportsOn",
		DestUID:    "local-cluster/019db722-d5c7-4085-9778-d8ebc33e95f2",
		DestKind:   "Pod",
	}
	AssertDeepEqual("edge", edge5, expectedEdge5, t)

	edge6 := edges[5]
	expectedEdge6 := Edge{
		SourceUID:  "local-cluster/53cd0e2e-34e0-454b-a0c4-e4dbf9306470",
		SourceKind: "PolicyReport",
		EdgeType:   "reportsOn",
		DestUID:    "local-cluster/019db722-d5c7-4085-9778-d8ebc33e95f2",
		DestKind:   "Pod",
	}
	AssertDeepEqual("edge", edge6, expectedEdge6, t)
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
