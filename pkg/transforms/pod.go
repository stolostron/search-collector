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
	nameSpace := p.Pod.Namespace

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
			glog.Errorf("ownedBy edge not created as node with ownerUID %s doesn't exist for pod %s", ownerUID, UID)
		}
	}

	//attachedTo edge
	secretMap := make(map[string]bool)
	configmapMap := make(map[string]bool)
	volumeClaimMap := make(map[string]bool)

	for _, container := range p.Pod.Spec.Containers {
		for _, envVal := range container.Env {
			if envVal.ValueFrom != nil {
				//If the env variable is a secret, add it to the map if it is not there
				if envVal.ValueFrom.SecretKeyRef != nil {
					secretName := envVal.ValueFrom.SecretKeyRef.Name
					//Check if the secretName already exists in secretMap
					if !secretMap[secretName] {
						secretMap[secretName] = true
					}
					//If the env variable is a ConfigMap, add it to the map if it is not there
				} else if envVal.ValueFrom.ConfigMapKeyRef != nil {
					configMapName := envVal.ValueFrom.ConfigMapKeyRef.Name
					if !configmapMap[configMapName] {
						configmapMap[configMapName] = true
					}
				}
			}
		}
	}
	for _, volume := range p.Pod.Spec.Volumes {
		if volume.Secret != nil {
			secretName := volume.Secret.SecretName
			if !secretMap[secretName] {
				secretMap[secretName] = true
			}
		} else if volume.ConfigMap != nil {
			configMapName := volume.ConfigMap.Name
			if !configmapMap[configMapName] {
				configmapMap[configMapName] = true
			}
		} else if volume.PersistentVolumeClaim != nil {
			volumeClaimName := volume.PersistentVolumeClaim.ClaimName
			if !volumeClaimMap[volumeClaimName] {
				volumeClaimMap[volumeClaimName] = true
			}
		}
	}
	var secrets []string
	for secret := range secretMap {
		secrets = append(secrets, secret)
	}
	var configmaps []string
	for configmap := range configmapMap {
		configmaps = append(configmaps, configmap)
	}
	var volumeClaims []string
	for volumeClaim := range volumeClaimMap {
		volumeClaims = append(volumeClaims, volumeClaim)
	}

	// Inner function used to get all edges for a specific destKind - the propLists are lists of resource names
	edgesByDestinationName := func(propList []string, destKind string) []Edge {
		attachedToEdges := []Edge{}
		if len(propList) > 0 {
			for _, name := range propList {
				if _, ok := ns.ByKindNamespaceName[destKind][nameSpace][name]; ok {
					attachedToEdges = append(attachedToEdges, Edge{
						SourceUID: UID,
						DestUID:   ns.ByKindNamespaceName[destKind][nameSpace][name].UID,
						EdgeType:  "attachedTo",
					})

				} else {
					glog.Errorf("attachedto edge not created as %s named %s not found", destKind, name)
				}
			}
		}
		return attachedToEdges
	}
	ret = append(ret, edgesByDestinationName(secrets, "Secret")...)
	ret = append(ret, edgesByDestinationName(configmaps, "ConfigMap")...)
	ret = append(ret, edgesByDestinationName(volumeClaims, "PersistentVolumeClaim")...)

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
			glog.Errorf("runsOn edge not created as node named %s of kind NODE not found", nodeName)
		}
	}
	return ret
}
