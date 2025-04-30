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
	"slices"
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

func getIsPolicyExternal(c *unstructured.Unstructured) bool {
	for _, m := range c.GetManagedFields() {
		if m.Manager == "multicluster-operators-subscription" ||
			strings.Contains(m.Manager, "argocd") {
			return true
		}
	}

	return false
}

// For cert, config, operator policies.
// This function returns `annotations`, `_isExternal` for `source`,
// and `severity`, `compliant`, and `remediationAction`.
func getPolicyCommonProperties(c *unstructured.Unstructured, node Node) Node {
	node.Properties["_isExternal"] = getIsPolicyExternal(c)

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

func OperatorPolicyResourceBuilder(c *unstructured.Unstructured) *PolicyResource {
	node := transformCommon(c) // Start off with the common properties
	node = getPolicyCommonProperties(c, node)
	node = recordRelatedObjects(c, node)

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

	return &PolicyResource{
		node: node,
	}
}

func ConfigPolicyResourceBuilder(c *unstructured.Unstructured) *PolicyResource {
	node := transformCommon(c) // Start off with the common properties
	node = getPolicyCommonProperties(c, node)
	node = recordRelatedObjects(c, node)

	return &PolicyResource{
		node: node,
	}
}

func recordRelatedObjects(c *unstructured.Unstructured, node Node) Node {
	relatedObjects, found, err := unstructured.NestedSlice(c.Object, "status", "relatedObjects")
	if !found || err != nil {
		return node
	}

	objList := make([]relatedObject, 0, len(relatedObjects))

	for _, item := range relatedObjects {
		relObj := parseConfigPolicyRelatedObject(item)
		if relObj == nil {
			continue
		}

		objList = append(objList, *relObj)
	}

	slices.SortFunc(objList, func(a, b relatedObject) int {
		return strings.Compare(a.kind+a.namespace+a.name, b.kind+b.namespace+b.name)
	})

	node.Metadata["relObjs"] = objList

	return node
}

func parseConfigPolicyRelatedObject(item any) *relatedObject {
	relObj, ok := item.(map[string]any)
	if !ok {
		return nil
	}

	object, found, err := unstructured.NestedMap(relObj, "object")
	if !found || err != nil {
		return nil
	}

	meta, found, err := unstructured.NestedStringMap(object, "metadata")
	if !found || err != nil {
		return nil
	}

	kind, ok := object["kind"].(string)
	if !ok {
		return nil
	}

	return &relatedObject{
		kind:      kind,
		namespace: meta["namespace"],
		name:      meta["name"],
	}
}

func CertPolicyResourceBuilder(c *unstructured.Unstructured) *PolicyResource {
	node := transformCommon(c) // Start off with the common properties

	detailMap, found, err := unstructured.NestedMap(c.Object, "status", "compliancyDetails")
	if len(detailMap) == 0 || !found || err != nil {
		return &PolicyResource{
			node: getPolicyCommonProperties(c, node),
		}
	}

	certList := []relatedObject{}
	for namespace, item := range detailMap {
		details, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// this "list" is actually a map
		nonCompCerts, found, err := unstructured.NestedMap(details, "nonCompliantCertificatesList")
		if len(nonCompCerts) == 0 || !found || err != nil {
			continue
		}

		for _, item := range nonCompCerts {
			cert, ok := item.(map[string]any)
			if !ok {
				continue
			}

			name, ok := cert["secretName"].(string)
			if !ok {
				continue
			}

			certList = append(certList, relatedObject{
				kind:      "Secret",
				namespace: namespace,
				name:      name,
			})
		}
	}

	slices.SortFunc(certList, func(a, b relatedObject) int {
		return strings.Compare(a.kind+a.namespace+a.name, b.kind+b.namespace+b.name)
	})

	node.Metadata["relObjs"] = certList

	return &PolicyResource{
		node: getPolicyCommonProperties(c, node),
	}
}

// BuildNode construct the node for Policy Resources
func (p PolicyResource) BuildNode() Node {
	return p.node
}

func (p PolicyResource) BuildEdges(ns NodeStore) []Edge {
	policyKind, ok := p.node.Properties["kind"].(string)
	if !ok {
		return []Edge{}
	}

	if relObjs, ok := p.node.Metadata["relObjs"].([]relatedObject); ok {
		info := NodeInfo{
			EdgeType: "relatedResource",
			Name:     p.node.Properties["name"].(string),
			UID:      p.node.UID,
			Kind:     policyKind,
		}

		return edgesFromRelatedObjects(info, ns, relObjs)
	}

	return []Edge{}
}
