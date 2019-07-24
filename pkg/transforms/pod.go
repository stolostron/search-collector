/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

type PodResource struct {
	*v1.Pod
}

func (pod PodResource) BuildNode() Node {
	// Loop over spec to get the container and image names
	var containers []string
	var images []string
	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
		images = append(images, container.Image)
	}

	// Loop over init container status or container status to get restarts and build status message
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	restarts := int64(0)
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
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
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = int64(0)
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

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

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	node := transformCommon(pod) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "Pod"
	node.Properties["hostIP"] = pod.Status.HostIP
	node.Properties["podIP"] = pod.Status.PodIP
	node.Properties["restarts"] = restarts
	node.Properties["status"] = reason
	node.Properties["container"] = containers
	node.Properties["image"] = images
	node.Properties["startedAt"] = ""
	if pod.Status.StartTime != nil {
		node.Properties["startedAt"] = pod.Status.StartTime.UTC().Format(time.RFC3339)
	}

	return node
}

func (p PodResource) BuildEdges(ns NodeStore) []Edge {
	ret := make([]Edge, 0, 8)

	//ownedBy edge
	ownerUID := ""
	UID := prefixedUID(p.Pod.UID)

	// Find the resource's owner. Resources can have multiple ownerReferences, but only one controller.
	for _, ref := range p.Pod.OwnerReferences {
		if *ref.Controller {
			ownerUID = prefixedUID(ref.UID) // TODO prefix with clustername
			continue
		}
	}

	//Lookup by UID to see if the owner Node exists
	if ownerUID != "" {
		if _, ok := ns.ByUID[ownerUID]; ok {
			ret = append(ret, Edge{
				SourceUID: UID,
				DestUID:   ownerUID,
				EdgeType:  "ownedBy",
			})
		} else {
			glog.V(2).Infof("Pod %s ownedBy edge not created: ownerUID %s not found", p.GetNamespace()+"/"+p.GetName(), ownerUID)
		}
	}

	//attachedTo edges
	secretMap := make(map[string]struct{})
	configmapMap := make(map[string]struct{})
	volumeClaimMap := make(map[string]struct{})

	for _, container := range p.Pod.Spec.Containers {
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
	for _, volume := range p.Pod.Spec.Volumes {
		if volume.Secret != nil {
			secretMap[volume.Secret.SecretName] = struct{}{}
		} else if volume.ConfigMap != nil {
			configmapMap[volume.ConfigMap.Name] = struct{}{}
		} else if volume.PersistentVolumeClaim != nil {
			volumeClaimMap[volume.PersistentVolumeClaim.ClaimName] = struct{}{}
		}
	}

	nodeInfo := NodeInfo{NameSpace: p.Namespace, UID: UID, EdgeType: "attachedTo", Kind: p.Kind}
	//Create all 'attachedTo' edges between pod and nodes of a specific kind(secrets, configmaps, volumeClaims)
	ret = append(ret, edgesByDestinationName(secretMap, ret, "Secret", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(configmapMap, ret, "ConfigMap", nodeInfo, ns)...)
	ret = append(ret, edgesByDestinationName(volumeClaimMap, ret, "PersistentVolumeClaim", nodeInfo, ns)...)

	//runsOn edges
	if p.Pod.Spec.NodeName != "" {
		nodeName := p.Pod.Spec.NodeName
		if _, ok := ns.ByKindNamespaceName["Node"]["_NONE"][nodeName]; ok {
			ret = append(ret, Edge{
				SourceUID: UID,
				DestUID:   ns.ByKindNamespaceName["Node"]["_NONE"][nodeName].UID,
				EdgeType:  "runsOn",
			})
		} else {
			glog.V(2).Infof("Pod %s runsOn edge not created: Node %s not found", p.GetNamespace()+"/"+p.GetName(), "_NONE/"+nodeName)
		}
	}
	return ret
}
