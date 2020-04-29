
package transforms

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)


type InsightResource struct {
	*unstructured.Unstructured
}

func (i InsightResource) BuildNode() Node {
	node := transformCommon(i)         // Start off with the common properties

	// description,_,_ := unstructured.NestedString(i, "spec.problem.description")
	// fmt.Println(">>>>> Building Insight.", description)

	node.Properties["kind"] = "Insight"
	node.Properties["group"] = "open-cluster-management.io"
	node.Properties["version"] = "v1alpha1"
	// apiGroupVersion(i.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type

	node.Properties["Description"] = "Problem description here."
	node.Properties["TopSolution"] = "Problem solution here."

	return node
}

func (i InsightResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
