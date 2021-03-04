// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	// "encoding/json"
	// "github.com/golang/glog"
	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyReport report
type PolicyReport struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Meta struct {
	// 	Name      string `json:"name"`
	// 	Namespace string `json:"namespace"`
	// } `json:"metadata"`
	Results []ReportResults `json:"results"`
}

// ReportResults rule violation results
type ReportResults struct {
	Policy string `json:"policy"`
	Message string `json:"message"`
	Category string `json:"category"`
	Status string `json:"status"`
	Data ReportData `json:"data"`
}

// ReportData rule violation data
type ReportData struct {
	Created string `json:"created_at"`
	Details string `json:"details"`
	TotalRisk string `json:"total_risk"`
	Reason string `json:"reason"`
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

	// Extract the properties specific to this type
	node.Properties["message"] = string(pr.Results[0].Message)
	node.Properties["category"] = strings.Split(pr.Results[0].Category, ",")
	node.Properties["risk"] = string(pr.Results[0].Data.TotalRisk)

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
