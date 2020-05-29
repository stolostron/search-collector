
package transforms

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)


type InsightResource struct {
	*unstructured.Unstructured
}

func (i InsightResource) BuildNode() Node {
	node := transformCommon(i)         // Start off with the common properties

	gvk := i.GroupVersionKind()
	node.Properties["kind"] = gvk.Kind
	node.Properties["apiversion"] = gvk.Version
	node.Properties["apigroup"] = gvk.Group

	// Extract the properties specific to this type

	description, _, _ := unstructured.NestedString(i.UnstructuredContent(), "spec", "problem", "description")
	node.Properties["description"] = description
	
	confidence, _, _ := unstructured.NestedInt(i.UnstructuredContent(), "spec", "problem", "confidence")
	node.Properties["confidence"] = confidence

	solutions, _, _ := unstructured.NestedSlice(i.UnstructuredContent(), "spec", "solutions")
	topSolution := solutions[0].(map[string]interface{})
	node.Properties["topsolution"] = topSolution["description"]

	resolution,_,_ := unstructured.NestedString(i.UnstructuredContent(), "spec", "resolution")
	node.Properties["resolution"] = resolution

	return node
}

func (i InsightResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
