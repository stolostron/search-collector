// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"sort"
	"strings"

	"github.com/stolostron/search-collector/pkg/config"
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
	Rule       string           `json:"rule,omitempty"`
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
	numRuleViolations := 0

	// Filter GRC sourced policy violations from node results
	// Policy details are displayed elsewhere in the UI, displaying them in the PR will result in double counts on pages
	results := []ReportResults{}
	for _, result := range pr.Results {
		// Include all the results for Kyverno generated policy reports
		if isKyverno || result.Source == "insights" {
			results = append(results, result)
		}

		if result.Result == "fail" || result.Result == "error" {
			numRuleViolations++
		}
	}

	// Total number of policies in the report
	node.Properties["numRuleViolations"] = numRuleViolations
	// Extract the properties specific to this type
	categoryMap := make(map[string]struct{})
	policies := sets.Set[string]{}
	rules := sets.Set[string]{}
	critical := 0
	important := 0
	moderate := 0
	low := 0

	policyViolationCounts := map[string]int{}

	for _, result := range results {
		for _, category := range strings.Split(result.Category, ",") {
			categoryMap[category] = struct{}{}
		}

		policies.Insert(result.Policy)
		if result.Rule != "" {
			rules.Insert(result.Rule)
		}

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

		if _, ok := policyViolationCounts[result.Policy]; !ok {
			policyViolationCounts[result.Policy] = 0
		}

		if result.Result == "fail" || result.Result == "error" {
			policyViolationCounts[result.Policy]++
		}
	}
	categories := make([]string, 0, len(categoryMap))
	for k := range categoryMap {
		categories = append(categories, k)
	}

	policyList := policies.UnsortedList()
	sort.Strings(policyList)

	ruleList := rules.UnsortedList()
	sort.Strings(ruleList)

	node.Properties["rules"] = ruleList
	node.Properties["policies"] = policyList
	node.Properties["category"] = categories
	node.Properties["critical"] = critical
	node.Properties["important"] = important
	node.Properties["moderate"] = moderate
	node.Properties["low"] = low
	// extract the cluster scope from the PolicyReport resource
	node.Properties["scope"] = string(pr.Scope.Name)
	node.Properties["_policyViolationCounts"] = policyViolationCounts
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

	// "policies" represents the policies
	for _, policy := range pr.node.Properties["policies"].([]string) {
		var kind, namespace, name string

		splitPolicy := strings.SplitN(policy, "/", 2)

		// Detect if it's a Policy or ClusterPolicy based on the presence of a namespace
		if len(splitPolicy) == 2 {
			kind = "Policy"
			namespace = splitPolicy[0]
			name = splitPolicy[1]
		} else {
			kind = "ClusterPolicy"
			namespace = "_NONE"
			name = policy
		}

		policyNode, ok := ns.ByKindNamespaceName[kind][namespace][name]
		if !ok {
			continue
		}

		edges = append(edges, Edge{
			SourceKind: policyNode.Properties["kind"].(string),
			SourceUID:  policyNode.UID,
			EdgeType:   "reports",
			DestKind:   pr.node.Properties["kind"].(string),
			DestUID:    pr.node.UID,
		})

		// The PolicyReport name is the UID of the violating object
		violatingObject, ok := ns.ByUID[config.Cfg.ClusterName+"/"+pr.node.Properties["name"].(string)]
		if !ok {
			continue
		}

		edges = append(edges, Edge{
			SourceKind: policyNode.Properties["kind"].(string),
			SourceUID:  policyNode.UID,
			EdgeType:   "appliesTo",
			DestUID:    violatingObject.UID,
			DestKind:   violatingObject.Properties["kind"].(string),
		}, Edge{
			SourceKind: pr.node.Properties["kind"].(string),
			SourceUID:  pr.node.UID,
			EdgeType:   "reportsOn",
			DestUID:    violatingObject.UID,
			DestKind:   violatingObject.Properties["kind"].(string),
		})
	}

	return edges
}
