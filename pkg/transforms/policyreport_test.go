// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("policyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	// Test unique fields that exist in policy report and are shown in UI - the common test will test the other bits
	AssertDeepEqual("category Length", len(node.Properties["category"].([]string)), 5, t)
	AssertDeepEqual("policies", node.Properties["policies"], []string{"insights/policyreport testing risk 1 policy", "insights/policyreport testing risk 2 policy"}, t)
	AssertDeepEqual("rules", len(node.Properties["rules"].([]string)), 0, t)
	AssertDeepEqual("numRuleViolations", node.Properties["numRuleViolations"], 2, t)
	AssertDeepEqual("critical", node.Properties["critical"], 0, t)
	AssertDeepEqual("important", node.Properties["important"], 0, t)
	AssertDeepEqual("moderate", node.Properties["moderate"], 1, t)
	AssertDeepEqual("low", node.Properties["low"], 1, t)

	AssertDeepEqual("scope", node.Properties["scope"], "test-cluster", t)
	expected := map[string]int{
		"insights/policyreport testing risk 1 policy": 1,
		"insights/policyreport testing risk 2 policy": 1,
	}
	AssertDeepEqual("_policyViolationCounts", node.Properties["_policyViolationCounts"].(map[string]int), expected, t)
}

func TestTransformKyvernoClusterPolicyReport(t *testing.T) {
	var pr PolicyReport
	UnmarshalFile("kyverno-clusterpolicyreport.json", &pr, t)
	node := PolicyReportResourceBuilder(&pr).BuildNode()

	AssertDeepEqual("apiversion", node.Properties["apiversion"].(string), "v1alpha2", t)
	AssertDeepEqual("category", node.Properties["category"].([]string), []string{"Kubecost"}, t)
	AssertDeepEqual("policies", node.Properties["policies"], []string{"ClusterPolicy/no-label-of-monkey", "ClusterPolicy/require-kubecost-labels"}, t)
	AssertDeepEqual("rules", node.Properties["rules"], []string{"no-monkey", "require-labels"}, t)
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
	AssertDeepEqual("apiversion", node.Properties["apiversion"].(string), "v1beta1", t)
	AssertDeepEqual(
		"policies",
		node.Properties["policies"],
		[]string{"ClusterPolicy/require-kubecost-labels", "NamespacedValidatingPolicy/test-namespace/require-kubecost-labels",
			"Policy/open-cluster-management-agent-addon/require-kubecost-labels", "ValidatingPolicy/require-kubecost-labels"},
		t,
	)
	AssertDeepEqual("rules", node.Properties["rules"], []string{"require-labels"}, t)
	AssertDeepEqual("numRuleViolations", node.Properties["numRuleViolations"], 4, t)
	expected := map[string]int{
		"require-kubecost-labels":                                           1,
		"open-cluster-management-agent-addon/require-kubecost-labels":       1,
		"ValidatingPolicy/require-kubecost-labels":                          1,
		"NamespacedValidatingPolicy/test-namespace/require-kubecost-labels": 1,
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

func TestKyvernoCELPolicyReportBuildEdges(t *testing.T) {
	validatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "ValidatingPolicy",
			"metadata": map[string]interface{}{
				"name": "require-kubecost-labels",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "272ec5b8-892b-40da-8b92-af141c377daa",
			},
		},
	}

	mutatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "MutatingPolicy",
			"metadata": map[string]interface{}{
				"name": "add-app-labels",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "373ec5b8-892b-40da-8b92-af141c377dbb",
			},
		},
	}

	imageValidatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "ImageValidatingPolicy",
			"metadata": map[string]interface{}{
				"name": "verify-image-signature",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "high",
				},
				"uid": "474ec5b8-892b-40da-8b92-af141c377dcc",
			},
		},
	}

	generatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "GeneratingPolicy",
			"metadata": map[string]interface{}{
				"name": "generate-network-policy",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "low",
				},
				"uid": "575ec5b8-892b-40da-8b92-af141c377ddd",
			},
		},
	}

	namespacedValidatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "NamespacedValidatingPolicy",
			"metadata": map[string]interface{}{
				"name":      "require-kubecost-labels",
				"namespace": "test-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "676ec5b8-892b-40da-8b92-af141c377eee",
			},
		},
	}

	namespacedMutatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "NamespacedMutatingPolicy",
			"metadata": map[string]interface{}{
				"name":      "add-app-labels",
				"namespace": "test-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "776ec5b8-892b-40da-8b92-af141c377fff",
			},
		},
	}

	namespacedImageValidatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "NamespacedImageValidatingPolicy",
			"metadata": map[string]interface{}{
				"name":      "verify-image-signature",
				"namespace": "test-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "high",
				},
				"uid": "876ec5b8-892b-40da-8b92-af141c378000",
			},
		},
	}

	namespacedGeneratingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "NamespacedGeneratingPolicy",
			"metadata": map[string]interface{}{
				"name":      "generate-network-policy",
				"namespace": "test-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "low",
				},
				"uid": "976ec5b8-892b-40da-8b92-af141c378111",
			},
		},
	}

	violatingPod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "open-cluster-management-agent-addon",
				"uid":       "019db722-d5c7-4085-9778-d8ebc33e95f2",
			},
		},
	}

	nodes := []Node{
		GenericResourceBuilder(validatingPolicy).node,
		GenericResourceBuilder(mutatingPolicy).node,
		GenericResourceBuilder(imageValidatingPolicy).node,
		GenericResourceBuilder(generatingPolicy).node,
		GenericResourceBuilder(namespacedValidatingPolicy).node,
		GenericResourceBuilder(namespacedMutatingPolicy).node,
		GenericResourceBuilder(namespacedImageValidatingPolicy).node,
		GenericResourceBuilder(namespacedGeneratingPolicy).node,
		GenericResourceBuilder(violatingPod).node,
	}
	nodeStore := BuildFakeNodeStore(nodes)

	var pr PolicyReport
	UnmarshalFile("kyverno-celpolicyreport.json", &pr, t)

	edges := PolicyReportResourceBuilder(&pr).BuildEdges(nodeStore)

	if len(edges) != 24 {
		t.Fatalf("Expected 24 edges but got %d", len(edges))
	}

	type edgeExpectation struct {
		edgeType   string
		sourceKind string
		destKind   string
		sourceUUID string
		destUUID   string
	}

	policyTests := []struct {
		nodeKind      string
		expectedEdges []edgeExpectation
	}{
		{
			nodeKind: "ValidatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "ValidatingPolicy", "Pod", "272ec5b8-892b-40da-8b92-af141c377daa", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "ValidatingPolicy", "PolicyReport", "272ec5b8-892b-40da-8b92-af141c377daa", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "MutatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "MutatingPolicy", "Pod", "373ec5b8-892b-40da-8b92-af141c377dbb", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "MutatingPolicy", "PolicyReport", "373ec5b8-892b-40da-8b92-af141c377dbb", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "ImageValidatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "ImageValidatingPolicy", "Pod", "474ec5b8-892b-40da-8b92-af141c377dcc", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "ImageValidatingPolicy", "PolicyReport", "474ec5b8-892b-40da-8b92-af141c377dcc", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "PolicyReport",
			expectedEdges: []edgeExpectation{
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reportsOn", "PolicyReport", "Pod", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
			},
		},
		{
			nodeKind: "GeneratingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "GeneratingPolicy", "Pod", "575ec5b8-892b-40da-8b92-af141c377ddd", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "GeneratingPolicy", "PolicyReport", "575ec5b8-892b-40da-8b92-af141c377ddd", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "NamespacedValidatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "NamespacedValidatingPolicy", "Pod", "676ec5b8-892b-40da-8b92-af141c377eee", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "NamespacedValidatingPolicy", "PolicyReport", "676ec5b8-892b-40da-8b92-af141c377eee", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "NamespacedMutatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "NamespacedMutatingPolicy", "Pod", "776ec5b8-892b-40da-8b92-af141c377fff", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "NamespacedMutatingPolicy", "PolicyReport", "776ec5b8-892b-40da-8b92-af141c377fff", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "NamespacedImageValidatingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "NamespacedImageValidatingPolicy", "Pod", "876ec5b8-892b-40da-8b92-af141c378000", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "NamespacedImageValidatingPolicy", "PolicyReport", "876ec5b8-892b-40da-8b92-af141c378000", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
		{
			nodeKind: "NamespacedGeneratingPolicy",
			expectedEdges: []edgeExpectation{
				{"appliesTo", "NamespacedGeneratingPolicy", "Pod", "976ec5b8-892b-40da-8b92-af141c378111", "019db722-d5c7-4085-9778-d8ebc33e95f2"},
				{"reports", "NamespacedGeneratingPolicy", "PolicyReport", "976ec5b8-892b-40da-8b92-af141c378111", "53cd0e2e-34e0-454b-a0c4-e4dbf9306470"},
			},
		},
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].SourceUID == edges[j].SourceUID {
			return edges[i].DestUID < edges[j].DestUID
		}

		return edges[i].SourceUID < edges[j].SourceUID
	})

	edgeIdx := 0
	for _, tc := range policyTests {
		for _, exp := range tc.expectedEdges {
			edge := edges[edgeIdx]
			edgeDesc := fmt.Sprintf("%s %s", tc.nodeKind, exp.edgeType)

			AssertEqual(edgeDesc+" type", string(edge.EdgeType), exp.edgeType, t)
			AssertEqual(edgeDesc+" source kind", edge.SourceKind, exp.sourceKind, t)
			AssertEqual(edgeDesc+" dest kind", edge.DestKind, exp.destKind, t)

			if exp.sourceUUID != "" && !strings.HasSuffix(edge.SourceUID, exp.sourceUUID) {
				t.Errorf("%s source UID should end with %s, got %s", edgeDesc, exp.sourceUUID, edge.SourceUID)
			}
			if exp.destUUID != "" && !strings.HasSuffix(edge.DestUID, exp.destUUID) {
				t.Errorf("%s dest UID should end with %s, got %s", edgeDesc, exp.destUUID, edge.DestUID)
			}

			edgeIdx++
		}
	}
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
