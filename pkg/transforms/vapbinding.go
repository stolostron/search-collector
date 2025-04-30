// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"encoding/json"
	"strings"

	"github.com/golang/glog"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type VapBindingResource struct {
	node Node
}

// Validating admission policy Binding
// Ref: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
func VapBindingResourceBuilder(v *unstructured.Unstructured) *VapBindingResource {
	node := transformCommon(v) // Start off with the common properties

	typeMeta := metav1.TypeMeta{
		Kind:       v.GetKind(),
		APIVersion: v.GetAPIVersion(),
	}

	apiGroupVersion(typeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type

	node.Properties["validationActions"], _, _ = unstructured.NestedStringSlice(v.Object, "spec", "validationActions")
	node.Properties["policyName"], _, _ = unstructured.NestedString(v.Object, "spec", "policyName")

	owners := v.GetOwnerReferences()

	for _, o := range owners {
		if strings.HasPrefix(o.APIVersion, "constraints.gatekeeper.sh") {
			node.Properties["_ownedBy"] = "Gatekeeper"

			break
		}

		if strings.HasPrefix(o.APIVersion, "kyverno.io") {
			node.Properties["_ownedBy"] = "Kyverno"

			break
		}
	}

	binding := &VapBindingResource{node: node}

	paramRefMap, ok, err := unstructured.NestedMap(v.Object, "spec", "paramRef")
	if !ok {
		return binding
	}

	if err != nil {
		glog.Errorf("Failed to parse the ValidatingAdmissionPolicyBinding %s paramRef: %v", v.GetName(), err)

		return binding
	}

	paramRefBytes, err := json.Marshal(paramRefMap)
	if err != nil {
		glog.Errorf("Failed to parse the ValidatingAdmissionPolicyBinding %s paramRef: %v", v.GetName(), err)

		return binding
	}

	paramRef := &admissionregistration.ParamRef{}

	err = json.Unmarshal(paramRefBytes, paramRef)
	if err != nil {
		glog.Errorf("Failed to parse the ValidatingAdmissionPolicyBinding %s paramRef: %v", v.GetName(), err)

		return binding
	}

	// Store this parameter to let the reconciler know to update the edges
	binding.node.Metadata["paramRef"] = paramRef

	return binding
}

// BuildNode construct the node for VapBindingResource
func (v VapBindingResource) BuildNode() Node {
	return v.node
}

// BuildEdges construct the edges for VapBindingResource
func (v VapBindingResource) BuildEdges(ns NodeStore) []Edge {
	policyName, ok := v.node.Properties["policyName"].(string)
	if !ok || policyName == "" {
		return []Edge{}
	}

	nodeInfo := NodeInfo{
		Name:      v.node.Properties["name"].(string),
		NameSpace: "_NONE",
		UID:       v.node.UID,
		EdgeType:  "attachedTo",
		Kind:      v.node.Properties["kind"].(string),
	}

	propSet := map[string]struct{}{policyName: {}}

	edges := edgesByDestinationName(propSet, "ValidatingAdmissionPolicy", nodeInfo, ns, []string{})

	if !v.node.hasMetadata("paramRef") {
		return edges
	}

	policy, ok := ns.ByKindNamespaceName["ValidatingAdmissionPolicy"]["_NONE"][policyName]
	if !ok {
		return edges
	}

	paramKind := policy.GetMetadata("paramKind_kind")
	if paramKind == "" {
		return edges
	}

	paramApiVersion := policy.GetMetadata("paramKind_apiVersion")
	if paramApiVersion == "" {
		return edges
	}

	paramGV, err := schema.ParseGroupVersion(paramApiVersion)
	if err != nil {
		return edges
	}

	namespaceToName := ns.ByKindNamespaceName[paramKind]

	paramRef, ok := v.node.Metadata["paramRef"].(*admissionregistration.ParamRef)
	if !ok || paramRef == nil {
		return edges
	}

	if paramRef.Namespace != "" {
		// If the namespace is specified, then limit the searches to just this namespace
		namespaceToName = map[string]map[string]Node{
			paramRef.Namespace: ns.ByKindNamespaceName[paramKind][paramRef.Namespace],
		}
	}

	var paramRefSelector labels.Selector

	if paramRef.Selector != nil {
		var err error

		paramRefSelector, err = metav1.LabelSelectorAsSelector(paramRef.Selector)
		if err != nil {
			return edges
		}
	}

	for _, nodeMap := range namespaceToName {
		for _, node := range nodeMap {
			version, ok := node.Properties["apiversion"].(string)
			if !ok || version != paramGV.Version {
				continue
			}

			group, ok := node.Properties["apigroup"].(string)
			if (!ok && group != "") || group != paramGV.Group {
				continue
			}

			if paramRef.Name != "" && paramRef.Name != node.Properties["name"].(string) {
				continue
			}

			if paramRefSelector != nil {
				nodeLabels, ok := node.Properties["label"].(map[string]string)
				if !ok {
					continue
				}

				nodeLabelsSet := labels.Set(nodeLabels)

				if !paramRefSelector.Matches(nodeLabelsSet) {
					continue
				}
			}

			edges = append(edges, Edge{
				EdgeType:   "paramReferences",
				SourceUID:  v.node.UID,
				SourceKind: "ValidatingAdmissionPolicyBinding",
				DestKind:   paramKind,
				DestUID:    node.UID,
			})
		}
	}

	return edges
}
