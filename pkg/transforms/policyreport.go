// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyReport report
type PolicyReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Results           []ReportResults `json:"results"`
}

// ReportResults rule violation results
type ReportResults struct {
	Policy     string           `json:"policy"`
	Message    string           `json:"message"`
	Category   string           `json:"category"`
	Result     string           `json:"result"`
	Properties ReportProperties `json:"properties"`
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

	// Total number of policies in the report
	node.Properties["numInsightPolicies"] = len(pr.Results)
	// Extract the properties specific to this type
	categoryMap := make(map[string]struct{})
	policies := make([]string, 0, len(pr.Results))
	for _, result := range pr.Results {
		for _, category := range strings.Split(result.Category, ",") {
			categoryMap[category] = struct{}{}
		}
		policies = append(policies, result.Policy)
	}
	categories := make([]string, 0, len(categoryMap))
	for k := range categoryMap {
		categories = append(categories, k)
	}
	node.Properties["insightPolicies"] = policies
	node.Properties["category"] = categories
	return &PolicyReportResource{node: node}
}

// BuildNode Creates the redisGraph node for this resource
func (pr PolicyReportResource) BuildNode() Node {
	return pr.node
}

// BuildEdges builds any necessary edges to related resources
func (pr PolicyReportResource) BuildEdges(ns NodeStore) []Edge {
	// TODO What edges does PolicyReport need
	ret := []Edge{}
	return ret
}
