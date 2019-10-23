/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type ServiceResource struct {
	*v1.Service
}

func (service ServiceResource) BuildNode() Node {
	node := transformCommon(service) // Start off with the common properties
	var ports []string
	apiGroupVersion(service.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["type"] = service.Spec.Type
	node.Properties["clusterIP"] = service.Spec.ClusterIP
	if len(service.Spec.ExternalIPs) > 0 {
		node.Properties["externalIPs"] = strings.Join(service.Spec.ExternalIPs, ",")
	}
	if len(service.Spec.Ports) > 0 {
		for _, p := range service.Spec.Ports {
			if p.NodePort != 0 {
				ports = append(ports, strings.Join([]string{strconv.Itoa(int(p.Port)), ":", strconv.Itoa(int(p.NodePort)), "/", string(p.Protocol)}, ""))
			} else {
				ports = append(ports, strings.Join([]string{strconv.Itoa(int(p.Port)), string(p.Protocol)}, "/"))
			}
		}
		node.Properties["port"] = ports
	}
	return node
}

func (s ServiceResource) BuildEdges(ns NodeStore) []Edge {
	serviceSelector := s.Spec.Selector

	if serviceSelector == nil {
		return []Edge{}
	}
	// TODO future: Match a pod in another namespace , but config will be different in those cases.
	pods := ns.ByKindNamespaceName["Pod"][s.Namespace]
	nodeInfo := NodeInfo{Name: s.Name, NameSpace: s.Namespace, UID: prefixedUID(s.UID), EdgeType: "usedBy", Kind: s.Kind}

	//Inner function to match the service and pod labels
	match := func(podLabels, serviceSelector map[string]string) bool {
		for selKey, selVal := range serviceSelector {
			if podVal, ok := podLabels[selKey]; podVal != selVal || !ok {
				return false
			}
		}
		return true
	}

	//usedBy edges
	ret := []Edge{}
	for _, p := range pods {
		if podLabels, ok := p.Properties["label"].(map[string]string); ok {
			if match(podLabels, serviceSelector) {
				ret = append(ret, edgesByOwner(p.UID, ns, nodeInfo)...)
			}
		}
	}

	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)

	return ret
}
