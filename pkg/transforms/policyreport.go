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
	policyViolationCounts := map[string]int{}
	critical := 0
	important := 0
	moderate := 0
	low := 0

	for _, result := range results {
		for _, category := range strings.Split(result.Category, ",") {
			categoryMap[category] = struct{}{}
		}

		policyName := result.Policy
		isNamespaced := strings.Contains(policyName, "/")
		kind := sourceToKind(result.Source, isNamespaced)

		// Policy keys match the format "kind/namespace/name" or "kind/name"
		policyKey := kind + "/" + policyName

		policies.Insert(policyKey)

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

		// For backwards compatibility, violation keys for Kyverno ClusterPolicy and Policy
		// are "name" and "namespace/name", respectively
		if kind == "ClusterPolicy" || kind == "Policy" {
			policyKey = policyName
		}

		if _, ok := policyViolationCounts[policyKey]; !ok {
			policyViolationCounts[policyKey] = 0
		}

		if result.Result == "fail" || result.Result == "error" {
			policyViolationCounts[policyKey]++
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

	policies := pr.node.Properties["policies"].([]string)

	// "policies" represents the policies
	for _, policy := range policies {
		var kind, namespace, name string

		parts := strings.SplitN(policy, "/", 3)

		switch len(parts) {
		case 3:
			kind = parts[0]
			namespace = parts[1]
			name = parts[2]
		case 2:
			kind = parts[0]
			namespace = "_NONE"
			name = parts[1]
		default:
			continue
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

// sourceToKind converts a report result source to its corresponding policy kind
func sourceToKind(source string, isNamespaced bool) string {
	// Kyverno sources defined in https://github.com/kyverno/kyverno/blob/main/pkg/utils/report/source.go

	switch source {
	case "kyverno":
		if isNamespaced {
			return "Policy"
		} else {
			return "ClusterPolicy"
		}
	case "KyvernoValidatingPolicy", "KyvernoImageValidatingPolicy", "KyvernoGeneratingPolicy", "KyvernoMutatingPolicy":
		trimmedSource := strings.TrimPrefix(source, "Kyverno")

		if isNamespaced {
			return "Namespaced" + trimmedSource
		} else {
			return trimmedSource
		}
	default:
		return source
	}
}
