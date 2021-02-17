/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

// PodResource ...
type PodResource struct {
	node Node
	Spec v1.PodSpec
}

// PodResourceBuilder ...
func PodResourceBuilder(p *v1.Pod) *PodResource {
	// Loop over spec to get the container and image names
	var containers []string
	var images []string
	for _, container := range p.Spec.Containers {
		containers = append(containers, container.Name)
		images = append(images, container.Image)
	}

	// Loop over init container status or container status to get restarts and build status message
	reason := string(p.Status.Phase)
	if p.Status.Reason != "" {
		reason = p.Status.Reason
	}

	initializing := false
	restarts := int64(0)
	for i := range p.Status.InitContainerStatuses {
		container := p.Status.InitContainerStatuses[i]
		restarts += int64(container.RestartCount)
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 &&
			container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(p.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = int64(0)
		hasRunning := false
		for i := len(p.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := p.Status.ContainerStatuses[i]

			restarts += int64(container.RestartCount)
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if p.DeletionTimestamp != nil && p.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if p.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	node := transformCommon(p) // Start off with the common properties

	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["hostIP"] = p.Status.HostIP
	node.Properties["podIP"] = p.Status.PodIP
	node.Properties["restarts"] = restarts
	node.Properties["status"] = reason
	node.Properties["container"] = containers
	node.Properties["image"] = images
	node.Properties["startedAt"] = ""
	if p.Status.StartTime != nil {
		node.Properties["startedAt"] = p.Status.StartTime.UTC().Format(time.RFC3339)
	}

	return &PodResource{node: node, Spec: p.Spec}
}

// BuildNode construct the node for the Pod Resources
func (p PodResource) BuildNode() Node {
	return p.node
}

// BuildEdges construct the edges for the Pod Resources
func (p PodResource) BuildEdges(ns NodeStore) []Edge {
	ret := make([]Edge, 0, 8)

	UID := p.node.UID

	nodeInfo := NodeInfo{
		Name:      p.node.Properties["name"].(string),
		NameSpace: p.node.Properties["namespace"].(string),
		UID:       UID,
		EdgeType:  "attachedTo",
		Kind:      p.node.Properties["kind"].(string)}

	//attachedTo edges
	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})
	volumeClaimMap := make(map[string]struct{})
	volumeMap := make(map[string]struct{})

	// Parse the pod's spec to create a list of all the secrets, configmaps and volumes it is attached to
	for _, container := range p.Spec.Containers {
		for _, envVal := range container.Env {
			if envVal.ValueFrom != nil {
				if envVal.ValueFrom.SecretKeyRef != nil {
					secretMap[envVal.ValueFrom.SecretKeyRef.Name] = struct{}{}
				} else if envVal.ValueFrom.ConfigMapKeyRef != nil {
					configmapMap[envVal.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
			}
		}
	}

	for _, volume := range p.Spec.Volumes {
		if volume.Secret != nil {
			secretMap[volume.Secret.SecretName] = struct{}{}
		} else if volume.ConfigMap != nil {
			configmapMap[volume.ConfigMap.Name] = struct{}{}
		} else if volume.PersistentVolumeClaim != nil {
			volumeClaimName := volume.PersistentVolumeClaim.ClaimName
			volumeClaimMap[volumeClaimName] = struct{}{}
			if pvClaimNode, ok := ns.ByKindNamespaceName["PersistentVolumeClaim"][nodeInfo.NameSpace][volumeClaimName]; ok {
				if volName, ok := pvClaimNode.Properties["volumeName"].(string); ok && pvClaimNode.Properties["volumeName"] != "" {
					volumeMap[volName] = struct{}{}
				}
			}
		}
	}

	//Create all 'attachedTo' edges between pod and nodes of a specific kind(secrets, configmaps, volumeClaims, volumes)
	ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns, []string{})...)
	ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns, []string{})...)
	ret = append(ret, edgesByDestinationName(volumeClaimMap, "PersistentVolumeClaim", nodeInfo, ns, []string{})...)
	nodeInfo.NameSpace = "_NONE"
	ret = append(ret, edgesByDestinationName(volumeMap, "PersistentVolume", nodeInfo, ns, []string{})...)

	//runsOn edges
	if p.Spec.NodeName != "" {
		nodeName := p.Spec.NodeName
		srcNode := ns.ByUID[UID]
		if dest, ok := ns.ByKindNamespaceName["Node"]["_NONE"][nodeName]; ok {
			if UID != dest.UID { //avoid connecting node to itself
				ret = append(ret, Edge{
					SourceUID:  UID,
					DestUID:    dest.UID,
					EdgeType:   "runsOn",
					SourceKind: srcNode.Properties["kind"].(string),
					DestKind:   dest.Properties["kind"].(string),
				})
			}
		} else {
			glog.V(2).Infof("Pod %s runsOn edge not created: Node %s not found",
				p.node.Properties["namespace"].(string)+"/"+p.node.Properties["name"].(string), "_NONE/"+nodeName)
		}
	}
	return ret
}
