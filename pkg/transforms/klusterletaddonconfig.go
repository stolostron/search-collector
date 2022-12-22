// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
)

// KlusterletAddonConfigResource ...
type KlusterletAddonConfigResource struct {
	node Node
}

// KlusterletAddonConfigResourceBuilder ...
func KlusterletAddonConfigResourceBuilder(p *agentv1.KlusterletAddonConfig) *KlusterletAddonConfigResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type
	enabledAddons := map[string]interface{}{}
	enabledAddons["search-collector"] = p.Spec.SearchCollectorConfig.Enabled
	enabledAddons["policy-controller"] = p.Spec.PolicyController.Enabled
	enabledAddons["cert-policy-controller"] = p.Spec.CertPolicyControllerConfig.Enabled
	enabledAddons["application-manager"] = p.Spec.ApplicationManagerConfig.Enabled
	enabledAddons["iam-policy-controller"] = p.Spec.IAMPolicyControllerConfig.Enabled
	node.Properties["addon"] = enabledAddons // maps to the enabled addons on the cluster

	return &KlusterletAddonConfigResource{node: node}
}

// BuildNode construct the nodes for the KlusterletAddonConfig Resources
func (p KlusterletAddonConfigResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the KlusterletAddonConfig Resources
func (p KlusterletAddonConfigResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
