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
	p "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
)

// PolicyResource ...
type PolicyResource struct {
	node Node
}

// PolicyResourceBuilder ...
func PolicyResourceBuilder(p *p.Policy) *PolicyResource {
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

// BuildNode construct the node for Policy Resources
func (p PolicyResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for Policy Resources
func (p PolicyResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
