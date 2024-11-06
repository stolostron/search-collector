// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KyvernoPolicyResource struct {
	node Node
}

// KyvernoPolicyResourceBuilder handles Kyverno Policy and ClusterPolicy objects. See:
//
//	https://github.com/kyverno/kyverno/blob/main/config/crds/kyverno/kyverno.io_policies.yaml
//	https://github.com/kyverno/kyverno/blob/main/config/crds/kyverno/kyverno.io_clusterpolicies.yaml
func KyvernoPolicyResourceBuilder(p *unstructured.Unstructured) *KyvernoPolicyResource {
	node := transformCommon(p) // Start off with the common properties
	apiGroupVersion(metav1.TypeMeta{Kind: p.GetKind(), APIVersion: p.GetAPIVersion()}, &node)
	node.Properties["_isExternal"] = getIsPolicyExternal(p)
	if validationFailureAction, ok, _ := unstructured.NestedString(p.Object, "spec", "validationFailureAction"); ok {
		node.Properties["validationFailureAction"] = validationFailureAction
	} else {
		// Audit is the default value and this makes the indexing easier
		node.Properties["validationFailureAction"] = "Audit"
	}

	if background, ok, _ := unstructured.NestedBool(p.Object, "spec", "background"); ok {
		node.Properties["background"] = background
	} else {
		// true is the default value and this makes the indexing easier
		node.Properties["background"] = true
	}

	if admission, ok, _ := unstructured.NestedBool(p.Object, "spec", "admission"); ok {
		node.Properties["admission"] = admission
	} else {
		// true is the default value and this makes the indexing easier
		node.Properties["admission"] = true
	}

	node.Properties["severity"] = p.GetAnnotations()["policies.kyverno.io/severity"]

	return &KyvernoPolicyResource{node: node}
}

func (p KyvernoPolicyResource) BuildNode() Node {
	return p.node
}

func (p KyvernoPolicyResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}
