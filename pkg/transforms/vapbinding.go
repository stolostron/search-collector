// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
type VapBindingResource struct {
	node Node
}

// Validating admission policy Binding
// Ref: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
func VapBindingResourceBuilder(v *unstructured.Unstructured) *VapBindingResource {
	node := transformCommon(v)         // Start off with the common properties

	typeMeta := metav1.TypeMeta{
		Kind:       v.GetKind(),
		APIVersion: v.GetAPIVersion(),
	}
	
	apiGroupVersion(typeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type

	node.Properties["validationActions"], _, _ = unstructured.NestedStringSlice(v.Object, "spec", "validationActions")
	node.Properties["policyName"], _, _ = unstructured.NestedString(v.Object, "spec", "policyName")

	owners := v.GetOwnerReferences()
	fromGK := false

	for _, o := range owners {
		if strings.HasPrefix(o.APIVersion, "constraints.gatekeeper.sh") {
			fromGK = true

			break
		}
	}

	node.Properties["_ownedByGatekeeper"] = fromGK

	return &VapBindingResource{node: node}
}

// BuildNode construct the node for VapBindingResource
func (v VapBindingResource) BuildNode() Node {
	return v.node
}

// BuildEdges construct the edges for VapBindingResource
func (v VapBindingResource) BuildEdges(ns NodeStore) []Edge {
	// no op for now to implement interface
	return []Edge{}
}
