// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const managedByLabel = "app.kubernetes.io/managed-by"

// PolicyReport report
type PolicyReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Results           []ReportResults        `json:"results"`
	Scope             corev1.ObjectReference `json:"scope"`
}

// ReportResults rule violation results
type ReportResults struct {
	Policy     string           `json:"policy"`
	Message    string           `json:"message"`
	Category   string           `json:"category"`
	Result     string           `json:"result"`
	Properties ReportProperties `json:"properties"`
	Source     string           `json:"source"`
}

// ReportProperties rule violation data
type ReportProperties struct {
	Created    string `json:"created_at"`
	Details    string `json:"details"`
	TotalRisk  string `json:"total_risk"`
	Reason     string `json:"reason"`
	Resolution string `json:"resolution"`
}

// PolicyReportResource type
type PolicyReportResource struct {
	node Node
}

// PolicyReportResourceBuilder ...
func PolicyReportResourceBuilder(pr *PolicyReport) *PolicyReportResource {
	node := transformCommon(pr) // Start off with the common properties

	gvk := pr.GroupVersionKind()
	node.Properties["kind"] = gvk.Kind
	node.Properties["apiversion"] = gvk.Version
	node.Properties["apigroup"] = gvk.Group

	isKyverno := pr.Labels[managedByLabel] == "kyverno"

	// Filter GRC sourced policy violations from node results
	// Policy details are displayed elsewhere in the UI, displaying them in the PR will result in double counts on pages
	results := []ReportResults{}
	for _, result := range pr.Results {
		if result.Source == "insights" || (isKyverno && (result.Result == "fail" || result.Result == "error")) {
			results = append(results, result)
		}
	}

	// Total number of policies in the report
	node.Properties["numRuleViolations"] = len(results)
	// Extract the properties specific to this type
	categoryMap := make(map[string]struct{})
	policies := sets.Set[string]{}
	critical := 0
	important := 0
	moderate := 0
	low := 0

	for _, result := range results {
		for _, category := range strings.Split(result.Category, ",") {
			categoryMap[category] = struct{}{}
		}
		policies.Insert(result.Policy)
		switch result.Properties.TotalRisk {
		case "4":
			critical++
		case "3":
			important++
		case "2":
			moderate++
		case "1":
			low++
		}
	}
	categories := make([]string, 0, len(categoryMap))
	for k := range categoryMap {
		categories = append(categories, k)
	}

	policyList := policies.UnsortedList()
	sort.Strings(policyList)

	// "rules" is incorrect since there is a "rule" field in the results, but this is kept for backwards compatibility
	node.Properties["rules"] = policyList
	node.Properties["policies"] = policyList
	node.Properties["category"] = categories
	node.Properties["critical"] = critical
	node.Properties["important"] = important
	node.Properties["moderate"] = moderate
	node.Properties["low"] = low
	// extract the cluster scope from the PolicyReport resource
	node.Properties["scope"] = string(pr.Scope.Name)
	return &PolicyReportResource{node: node}
}

// BuildNode Creates the redisGraph node for this resource
func (pr PolicyReportResource) BuildNode() Node {
	return pr.node
}

// BuildEdges builds any necessary edges to related resources
func (pr PolicyReportResource) BuildEdges(ns NodeStore) []Edge {
	edges := []Edge{}

	labels, ok := pr.node.Properties["label"].(map[string]string)
	if !ok || labels[managedByLabel] != "kyverno" {
		return edges
	}

	nodeInfo := NodeInfo{
		Name:     pr.node.Properties["name"].(string),
		UID:      pr.node.UID,
		EdgeType: "reportedBy",
		Kind:     pr.node.Properties["kind"].(string),
	}

	if nodeInfo.Kind == "PolicyReport" {
		nodeInfo.NameSpace = pr.node.Properties["namespace"].(string)
	} else if nodeInfo.Kind == "ClusterPolicyReport" {
		nodeInfo.NameSpace = "_NONE"
	}

	policyKindToPolicies := map[string]map[string]struct{}{}

	for _, policy := range pr.node.Properties["policies"].([]string) {
		splitPolicy := strings.SplitN(policy, "/", 2)

		// Detect if it's a Policy or ClusterPolicy based on the presence of a namespace
		if len(splitPolicy) == 2 {
			if policyKindToPolicies["Policy"] == nil {
				policyKindToPolicies["Policy"] = map[string]struct{}{}
			}

			policyKindToPolicies["Policy"][policy] = struct{}{}
		} else {
			if policyKindToPolicies["ClusterPolicy"] == nil {
				policyKindToPolicies["ClusterPolicy"] = map[string]struct{}{}
			}

			// Explicitly set cluster scope since a ClusterPolicy can generate a namespaced PolicyReport.
			policyKindToPolicies["ClusterPolicy"]["_NONE/"+policy] = struct{}{}
		}
	}

	for kind, propSet := range policyKindToPolicies {
		edges = append(edges, edgesByDestinationName(propSet, kind, nodeInfo, ns, []string{})...)
	}

	return edges
}
