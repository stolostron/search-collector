package transforms

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type InsightResource struct {
	*unstructured.Unstructured
}

func (i InsightResource) BuildNode() Node {
	node := transformCommon(i) // Start off with the common properties

	gvk := i.GroupVersionKind()
	node.Properties["kind"] = gvk.Kind
	node.Properties["apiversion"] = gvk.Version
	node.Properties["apigroup"] = gvk.Group

	// Extract the properties specific to this type

	description, _, _ := unstructured.NestedString(i.UnstructuredContent(), "spec", "problem", "description")
	node.Properties["description"] = description

	confidence, _, _ := unstructured.NestedInt64(i.UnstructuredContent(), "spec", "problem", "confidence")
	node.Properties["confidence"] = confidence

	solutions, _, _ := unstructured.NestedSlice(i.UnstructuredContent(), "spec", "solutions")
	topSolution := solutions[0].(map[string]interface{})
	node.Properties["topsolution"] = topSolution["description"]

	resolution, _, _ := unstructured.NestedString(i.UnstructuredContent(), "spec", "resolution")
	node.Properties["resolution"] = resolution

	return node
}

func (i InsightResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(i.GetUID())

	nodeInfo := NodeInfo{NameSpace: i.GetNamespace(), UID: UID, EdgeType: "relatedTo", Kind: i.GetKind(), Name: i.GetName()}
	// var podNode Node

	//add edges to all owners
	//Excluding controller as edge to controller is built in commonEdges
	for _, owner := range i.GetOwnerReferences() {
		if owner.Controller != nil && !*owner.Controller {
			edge := Edge{SourceKind: nodeInfo.Kind, DestKind: owner.Kind, SourceUID: nodeInfo.UID, DestUID: string(owner.UID), EdgeType: nodeInfo.EdgeType}
			ret = append(ret, edge)
		}
	}

	//add attachedTo edges using insight labels
	nodeInfo.EdgeType = "attachedTo"

	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})
	volumeClaimMap := make(map[string]struct{})
	volumeMap := make(map[string]struct{})

	labels := i.GetLabels()
	secrets, ok := labels["secrets"]
	if ok {
		for _, secret := range strings.Split(secrets, ", ") {
			secretMap[secret] = struct{}{}
		}
	}
	configmaps, ok := labels["configmaps"]
	if ok {
		for _, configmap := range strings.Split(configmaps, ", ") {
			configmapMap[configmap] = struct{}{}
		}
	}
	volumeClaims, ok := labels["volumeClaims"]
	if ok {
		for _, volumeClaim := range strings.Split(volumeClaims, ", ") {
			volumeClaimMap[volumeClaim] = struct{}{}
			if pvClaimNode, ok := ns.ByKindNamespaceName["PersistentVolumeClaim"][nodeInfo.NameSpace][volumeClaim]; ok {
				if volName, ok := pvClaimNode.Properties["volumeName"].(string); ok && pvClaimNode.Properties["volumeName"] != "" {
					volumeMap[volName] = struct{}{}
				}
			}
		}
	}
	//Create all 'attachedTo' edges between insight and nodes of a specific kind(secrets, configmaps, volumeClaims, volumes)
	ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(volumeClaimMap, "PersistentVolumeClaim", nodeInfo, ns)...)
	nodeInfo.NameSpace = "_NONE"
	ret = append(ret, edgesByDestinationName(volumeMap, "PersistentVolume", nodeInfo, ns)...)

	return ret
}
