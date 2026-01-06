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
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ServiceResource ...
type ServiceResource struct {
	node Node
	Spec v1.ServiceSpec
}

// ServiceResourceBuilder ...
func ServiceResourceBuilder(s *v1.Service, r *unstructured.Unstructured) *ServiceResource {
	node := transformCommon(s) // Start off with the common properties
	var ports []string
	apiGroupVersion(s.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["type"] = s.Spec.Type
	node.Properties["clusterIP"] = s.Spec.ClusterIP
	if len(s.Spec.ExternalIPs) > 0 {
		node.Properties["externalIPs"] = strings.Join(s.Spec.ExternalIPs, ",")
	}
	if len(s.Spec.Ports) > 0 {
		for _, p := range s.Spec.Ports {
			if p.NodePort != 0 {
				ports = append(ports, strings.Join([]string{strconv.Itoa(int(p.Port)), ":",
					strconv.Itoa(int(p.NodePort)), "/", string(p.Protocol)}, ""))
			} else {
				ports = append(ports, strings.Join([]string{strconv.Itoa(int(p.Port)), string(p.Protocol)}, "/"))
			}
		}
		node.Properties["port"] = ports
	}

	node = applyDefaultTransformConfig(node, r)

	return &ServiceResource{node: node, Spec: s.Spec}
}

// BuildNode construct the node for the Service Resources
func (s ServiceResource) BuildNode() Node {
	return s.node
}

// BuildEdges construct the edges for the Service Resources
func (s ServiceResource) BuildEdges(ns NodeStore) []Edge {
	serviceSelector := s.Spec.Selector

	if serviceSelector == nil {
		return []Edge{}
	}

	// Future: Match a pod in another namespace , but config will be different in those cases.
	pods := ns.ByKindNamespaceName["Pod"][s.node.Properties["namespace"].(string)]
	nodeInfo := NodeInfo{
		Name:      s.node.Properties["name"].(string),
		NameSpace: s.node.Properties["namespace"].(string),
		UID:       s.node.UID,
		EdgeType:  "usedBy",
		Kind:      s.node.Properties["kind"].(string)}

	// Inner function to match the service and pod labels
	match := func(podLabels, serviceSelector map[string]string) bool {
		for selKey, selVal := range serviceSelector {
			if podVal, ok := podLabels[selKey]; podVal != selVal || !ok {
				return false
			}
		}
		return true
	}

	// usedBy edges
	ret := []Edge{}
	for _, p := range pods {
		if podLabels, ok := p.Properties["label"].(map[string]string); ok {
			if match(podLabels, serviceSelector) {
				ret = append(ret, edgesByOwner(p.UID, ns, nodeInfo, []string{})...)
			}
		}
	}

	return ret
}
