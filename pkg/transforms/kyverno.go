// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

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

	if strings.HasPrefix(p.GetAPIVersion(), "policies.kyverno.io") {
		node.Properties["validationFailureAction"] = getValidationAction(p.Object)
	} else {
		node.Properties["validationFailureAction"] = getConsolidatedFailureAction(p.Object)
	}

	// true is the default value and this makes the indexing easier
	background := true
	admission := true

	evaluationMap, hasEvaluation, _ := unstructured.NestedMap(p.Object, "spec", "evaluation")

	if hasEvaluation { // for policies.kyverno.io policies
		if bg, ok, _ := unstructured.NestedBool(evaluationMap, "background", "enabled"); ok {
			background = bg
		}

		if admit, ok, _ := unstructured.NestedBool(evaluationMap, "admission", "enabled"); ok {
			admission = admit
		}
	} else { // for kyverno.io policies
		if bg, ok, _ := unstructured.NestedBool(p.Object, "spec", "background"); ok {
			background = bg
		}

		if admit, ok, _ := unstructured.NestedBool(p.Object, "spec", "admission"); ok {
			admission = admit
		}
	}

	node.Properties["background"] = background
	node.Properties["admission"] = admission
	node.Properties["severity"] = p.GetAnnotations()["policies.kyverno.io/severity"]

	return &KyvernoPolicyResource{node: node}
}

func (p KyvernoPolicyResource) BuildNode() Node {
	return p.node
}

func (p KyvernoPolicyResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}

func getValidationFailureAction(clusterPolicy map[string]interface{}) string {
	// Audit is the default value and this makes the indexing easier
	// NOTE This validationFailureAction is deprecated as of v Kyverno 1.13.
	// Scheduled to be removed in a future version.
	if validationFailureAction, ok, _ := unstructured.NestedString(clusterPolicy,
		"spec", "validationFailureAction"); ok {
		return validationFailureAction
	} else {
		return "Audit"
	}
}

// getConsolidatedFailureAction returns the failure action for
// a Kyverno ClusterPolicy or Policy
func getConsolidatedFailureAction(clusterPolicy map[string]interface{}) string {
	rules, ok, _ := unstructured.NestedSlice(clusterPolicy, "spec", "rules")
	if !ok {
		return getValidationFailureAction(clusterPolicy)
	}

	actions := ""

	for _, r := range rules {
		mapRule, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		failureAction, ok, _ := unstructured.NestedString(mapRule, "validate", "failureAction")
		if !ok {
			continue
		}

		// The failureAction in a rules takes priority over validationFailureAction in spec.
		if actions == "" {
			actions = failureAction
		} else if actions != failureAction {
			return "Audit/Enforce"
		}
	}

	if actions == "" {
		return getValidationFailureAction(clusterPolicy)
	}

	return actions
}

// getValidationAction returns the policy enforcement action for
// Kyverno policies.kyverno.io/v1
func getValidationAction(policy map[string]interface{}) string {
	validationActions, ok, _ := unstructured.NestedSlice(policy, "spec", "validationActions")
	if !ok {
		return "" // Some policy types do not have any validation actions
	}

	hasAudit := false
	hasDeny := false

	for _, action := range validationActions {
		if actionStr, ok := action.(string); ok {
			switch actionStr {
			case "Audit":
				hasAudit = true
			case "Deny":
				hasDeny = true
			}
		}
	}

	switch {
	case hasAudit && hasDeny:
		return "Audit/Deny"
	case hasDeny:
		return "Deny"
	case hasAudit:
		return "Audit"
	}
	// "Warn" is not collected because the ClusterPolicy equivalent
	// "emitWarning" is not collected
	return ""
}
