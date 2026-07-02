// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strconv"

	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// KlusterletAddonConfigResource ...
type KlusterletAddonConfigResource struct {
	node Node
}

// KlusterletAddonConfigResourceBuilder ...
func KlusterletAddonConfigResourceBuilder(p *agentv1.KlusterletAddonConfig, r *unstructured.Unstructured,
	additionalColumns ...ExtractProperty) *KlusterletAddonConfigResource {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type
	enabledAddons := map[string]interface{}{}
	enabledAddons["search-collector"] = strconv.FormatBool(p.Spec.SearchCollectorConfig.Enabled)
	enabledAddons["policy-controller"] = strconv.FormatBool(p.Spec.PolicyController.Enabled)
	enabledAddons["cert-policy-controller"] = strconv.FormatBool(p.Spec.CertPolicyControllerConfig.Enabled)
	enabledAddons["application-manager"] = strconv.FormatBool(p.Spec.ApplicationManagerConfig.Enabled)
	enabledAddons["iam-policy-controller"] = strconv.FormatBool(p.Spec.IAMPolicyControllerConfig.Enabled)
	node.Properties["addon"] = enabledAddons // maps to the enabled addons on the cluster

	node = applyDefaultTransformConfig(node, r, additionalColumns...)
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
