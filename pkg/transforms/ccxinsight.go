package transforms

import (
	"encoding/json"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PolicyReport report
type PolicyReport struct {
	Meta struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
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

type CCXInsightResource struct {
	*unstructured.Unstructured
}

func (i CCXInsightResource) BuildNode() Node {
	node := transformCommon(i) // Start off with the common properties

	gvk := i.GroupVersionKind()
	node.Properties["kind"] = gvk.Kind
	node.Properties["apiversion"] = gvk.Version
	node.Properties["apigroup"] = gvk.Group

	// Extract the properties specific to this type
	reportData, _, _ := unstructured.NestedMap(i.UnstructuredContent())
	data, _ := json.Marshal(reportData)
	var policyReport PolicyReport
	err := json.Unmarshal(data, &policyReport)
	if err != nil {
		glog.Fatal("Unable to unmarshal policy report json ", err)
	}

	node.Properties["message"] = policyReport.Results[0].Message
	node.Properties["category"] = policyReport.Results[0].Category
	node.Properties["risk"] = policyReport.Results[0].Data.TotalRisk

	return node
}

func (i CCXInsightResource) BuildEdges(ns NodeStore) []Edge {
	// TODO does the PolicyReport have edges?
	ret := []Edge{}
	return ret
}
