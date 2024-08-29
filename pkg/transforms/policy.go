/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

	policy "github.com/stolostron/governance-policy-propagator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PolicyResource ...
type PolicyResource struct {
	node Node
}

// PolicyResourceBuilder ...
func PolicyResourceBuilder(p *policy.Policy) *PolicyResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["remediationAction"] = string(p.Spec.RemediationAction)
	node.Properties["disabled"] = p.Spec.Disabled
	node.Properties["numRules"] = len(p.Spec.PolicyTemplates)
	// For the root policy (on hub, in non cluster ns), it doesn’t have an overall status. it has status per cluster.
	// On managed cluster, compliance is reported by status.compliant
	if p.Status.ComplianceState != "" {
		node.Properties["compliant"] = string(p.Status.ComplianceState)
	}
	pnamespace, okns := p.ObjectMeta.Labels["parent-namespace"]
	ppolicy, okpp := p.ObjectMeta.Labels["parent-policy"]
	if okns && okpp {
		node.Properties["_parentPolicy"] = pnamespace + "/" + ppolicy
	}

	return &PolicyResource{node: node}
}

// For cert, config, operator policies.
// This function returns `annotations`, `_isExternal` for `source`,
// and `severity`, `compliant`, and `remediationAction`.
func getPolicyCommonProperties(c *unstructured.Unstructured, node Node) Node {
	node.Properties["_isExternal"] = false

	for _, m := range c.GetManagedFields() {
		if m.Manager == "multicluster-operators-subscription" ||
			strings.Contains(m.Manager, "argocd") {
			node.Properties["_isExternal"] = true

			break
		}
	}

	typeMeta := metav1.TypeMeta{
		Kind:       c.GetKind(),
		APIVersion: c.GetAPIVersion(),
	}

	apiGroupVersion(typeMeta, &node) // add kind, apigroup and version

	node.Properties["compliant"], _, _ = unstructured.NestedString(c.Object, "status", "compliant")

	node.Properties["severity"], _, _ = unstructured.NestedString(c.Object, "spec", "severity")

	node.Properties["remediationAction"], _, _ = unstructured.NestedString(c.Object, "spec", "remediationAction")

	node.Properties["disabled"], _, _ = unstructured.NestedBool(c.Object, "spec", "disabled")

	return node
}

// For cert, config policies.
func StandalonePolicyResourceBuilder(c *unstructured.Unstructured) *PolicyResource {
	node := transformCommon(c) // Start off with the common properties

	return &PolicyResource{node: getPolicyCommonProperties(c, node)}
}

func OperatorPolicyResourceBuilder(c *unstructured.Unstructured) *PolicyResource {
	node := transformCommon(c) // Start off with the common properties

	node = getPolicyCommonProperties(c, node)

	var deploymentAvailable bool
	var upgradeAvailable bool

	conditions, _, _ := unstructured.NestedSlice(c.Object, "status", "conditions")
	for _, condition := range conditions {
		mapCondition, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		conditionType, found, err := unstructured.NestedString(mapCondition, "type")
		if !found || err != nil {
			continue
		}

		conditionReason, found, err := unstructured.NestedString(mapCondition, "reason")
		if !found || err != nil {
			continue
		}

		if conditionType == "InstallPlanCompliant" {
			if conditionReason == "InstallPlanRequiresApproval" || conditionReason == "InstallPlanApproved" {
				upgradeAvailable = true
			} else {
				upgradeAvailable = false
			}
		} else if conditionType == "DeploymentCompliant" {
			if conditionReason == "DeploymentsAvailable" {
				deploymentAvailable = true
			} else {
				deploymentAvailable = false
			}
		}
	}

	node.Properties["deploymentAvailable"] = deploymentAvailable
	node.Properties["upgradeAvailable"] = upgradeAvailable

	return &PolicyResource{node: node}
}

// BuildNode construct the node for Policy Resources
func (p PolicyResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for Policy Resources
func (p PolicyResource) BuildEdges(ns NodeStore) []Edge {
	// no op for now to implement interface
	return []Edge{}
}
