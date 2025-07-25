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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PersistentVolumeClaimResource ...
type PersistentVolumeClaimResource struct {
	node Node
}

// PersistentVolumeClaimResourceBuilder ...
func PersistentVolumeClaimResourceBuilder(p *v1.PersistentVolumeClaim, r *unstructured.Unstructured,
	additionalColumns ...ExtractProperty) *PersistentVolumeClaimResource {
	node := transformCommon(p) // Start off with the common properties

	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["status"] = string(p.Status.Phase)
	node.Properties["volumeName"] = string(p.Spec.VolumeName)
	if p.Spec.StorageClassName != nil {
		node.Properties["storageClassName"] = *p.Spec.StorageClassName
	}
	node.Properties["capacity"] = ""
	storage, ok := p.Status.Capacity["storage"]
	if ok {
		node.Properties["capacity"] = storage.String()
	}
	// can't cast []PersistentVolumeClaimAccessMode to []string without unsafe
	accessModes := make([]string, len(p.Spec.AccessModes))
	for i := 0; i < len(p.Spec.AccessModes); i++ {
		accessModes[i] = string(p.Spec.AccessModes[i])
	}
	node.Properties["accessMode"] = accessModes

	if p.Spec.Resources.Requests != nil {
		request, ok := p.Spec.Resources.Requests["storage"]
		if ok {
			node.Properties["request"] = request.String()
		}
	}

	node = applyDefaultTransformConfig(node, r, additionalColumns...)

	return &PersistentVolumeClaimResource{node: node}
}

// BuildNode construct the node for the PersistentVolumeClaim Resources
func (p PersistentVolumeClaimResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the PersistentVolumeClaim Resources
func (p PersistentVolumeClaimResource) BuildEdges(ns NodeStore) []Edge {
	ret := make([]Edge, 0, 8)
	UID := p.node.UID
	pvClaimNode := ns.ByUID[UID]
	//boundTo edges
	nodeInfo := NodeInfo{
		Name:      p.node.Properties["name"].(string),
		NameSpace: "_NONE",
		UID:       UID,
		EdgeType:  "boundTo",
		Kind:      p.node.Properties["kind"].(string)}

	volumeMap := make(map[string]struct{})
	if pvClaimNode.Properties["volumeName"] != "" {
		if volName, ok := pvClaimNode.Properties["volumeName"].(string); ok {
			volumeMap[volName] = struct{}{}
			ret = append(ret, edgesByDestinationName(volumeMap, "PersistentVolume", nodeInfo, ns, []string{})...)
		}
	}
	return ret
}
