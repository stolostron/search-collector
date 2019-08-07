/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strings"
	"time"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiTypes "k8s.io/apimachinery/pkg/types"
)

// An object given to the Edge Building methods in the transforms package.
// Contains representations of the Node list that are useful for them to efficiently find the nodes that they need.
type NodeStore struct {
	ByUID               map[string]Node
	ByKindNamespaceName map[string]map[string]map[string]Node
}

// Extracts the common properties from a default k8s resource of unknown type and returns them in a map ready to be put in an Node
func commonProperties(resource machineryV1.Object) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["name"] = resource.GetName()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().UTC().Format(time.RFC3339)
	ret["_clusterNamespace"] = config.Cfg.ClusterNamespace
	if config.Cfg.DeployedInHub {
		ret["_hubClusterResource"] = true
	}

	if resource.GetLabels() != nil {
		ret["label"] = resource.GetLabels()
	}
	if resource.GetNamespace() != "" {
		ret["namespace"] = resource.GetNamespace()
	}

	if resource.GetAnnotations()["app.ibm.com/hosting-subscription"] != "" {
		ret["_app.ibm.com/hosting-subscription"] = resource.GetAnnotations()["app.ibm.com/hosting-subscription"]
	}
	if resource.GetAnnotations()["app.ibm.com/hosting-deployable"] != "" {
		ret["_app.ibm.com/hosting-deployable"] = resource.GetAnnotations()["app.ibm.com/hosting-deployable"]
	}
	return ret
}

// Transforms a resource of unknown type by simply pulling out the common properties.
func transformCommon(resource machineryV1.Object) Node {
	n := Node{
		UID:        prefixedUID(resource.GetUID()),
		Properties: commonProperties(resource),
		Metadata:   make(map[string]string),
	}
	n.Metadata["OwnerUID"] = ownerRefUID(resource.GetOwnerReferences())
	return n
}

// Extracts the properties from a non-default k8s resource and returns them in a map ready to be put in an Node
func unstructuredProperties(resource UnstructuredResource) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["kind"] = resource.GetKind()
	ret["name"] = resource.GetName()
	ret["selfLink"] = resource.GetSelfLink()
	ret["created"] = resource.GetCreationTimestamp().UTC().Format(time.RFC3339)
	ret["_clusterNamespace"] = config.Cfg.ClusterNamespace
	if config.Cfg.DeployedInHub {
		ret["_hubClusterResource"] = true
	}

	// valid api group with have format of "apigroup/version"
	// unnamed api groups will have format of "/version"
	slice := strings.Split(resource.GetAPIVersion(), "/")
	if len(slice) == 2 {
		ret["apigroup"] = slice[0]
	}

	if resource.GetLabels() != nil {
		ret["label"] = resource.GetLabels()
	}
	if resource.GetNamespace() != "" {
		ret["namespace"] = resource.GetNamespace()
	}
	if resource.GetAnnotations()["app.ibm.com/hosting-subscription"] != "" {
		ret["_app.ibm.com/hosting-subscription"] = resource.GetAnnotations()["app.ibm.com/hosting-subscription"]
	}
	if resource.GetAnnotations()["app.ibm.com/hosting-deployable"] != "" {
		ret["_app.ibm.com/hosting-deployable"] = resource.GetAnnotations()["app.ibm.com/hosting-deployable"]
	}
	return ret

}

type UnstructuredResource struct {
	*unstructured.Unstructured
}

func (u UnstructuredResource) BuildNode() Node {
	n := Node{
		UID:        prefixedUID(u.GetUID()),
		Properties: unstructuredProperties(u),
		Metadata:   make(map[string]string),
	}
	n.Metadata["OwnerUID"] = ownerRefUID(u.GetOwnerReferences())
	return n
}

func (u UnstructuredResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}

// Prefixes the given UID with the cluster name from config and a /
func prefixedUID(uid apiTypes.UID) string {
	return strings.Join([]string{config.Cfg.ClusterName, string(uid)}, "/")
}

// Prefixes the given UID with the cluster name from config and a /
func ownerRefUID(ownerReferences []machineryV1.OwnerReference) string {
	ownerUID := ""
	for _, ref := range ownerReferences {
		if ref.Controller != nil && *ref.Controller {
			ownerUID = prefixedUID(ref.UID)
			continue
		}
	}
	return ownerUID
}

type NodeInfo struct {
	EdgeType
	Name, NameSpace, UID, Kind string
}

// Function to create an edge between the pod and it's owner, if it exists
// If the pod is owned by a replicaset which in turn is owned by a deployment, the function will be recursively called to create edges between pod->replicaset and pod->deployment
func edgesByOwner(destUID string, ownedByEdges []Edge, ns NodeStore, nodeInfo NodeInfo) []Edge {
	if destUID != "" {
		//Lookup by UID to see if the owner Node exists
		if dest, ok := ns.ByUID[destUID]; ok {
			ownedByEdges = append(ownedByEdges, Edge{
				SourceUID: nodeInfo.UID,
				DestUID:   destUID,
				EdgeType:  nodeInfo.EdgeType,
			})

			if dest.GetMetadata("ReleaseUID") != "" { // If owner included/owned by a release...
				if _, ok := ns.ByUID[dest.GetMetadata("ReleaseUID")]; ok { // ...make sure the release exists...
					ownedByEdges = append(ownedByEdges, Edge{ // ... then add edge from source to release
						SourceUID: nodeInfo.UID,
						DestUID:   dest.GetMetadata("ReleaseUID"),
						EdgeType:  "ownedBy",
					})
				}
			}

			// If the destination node has property _ownerUID, create an edge between the pod and the destination's owner
			// Call the edgesByOwner recursively to create the ownedBy edge
			if dest.GetMetadata("OwnerUID") != "" {
				ownedByEdges = append(ownedByEdges, edgesByOwner(dest.GetMetadata("OwnerUID"), ownedByEdges, ns, nodeInfo)...)
			}

		} else {
			glog.V(2).Infof("For %s, %s, %s edge not created: ownerUID %s not found", nodeInfo.Kind, nodeInfo.NameSpace+"/"+nodeInfo.Name, nodeInfo.EdgeType, destUID)
		}
	}
	return ownedByEdges
}

// Function used to get all edges for a specific destKind - the propSet are maps of resource names, nodeInfo has additional info about the node and nodestore has all the current nodes
func edgesByDestinationName(propSet map[string]struct{}, attachedToEdges []Edge, destKind string, nodeInfo NodeInfo, ns NodeStore) []Edge {

	if len(propSet) > 0 {
		for name := range propSet {
			// For channels, get the channel namespace and name from each string
			if destKind == "Channel" {
				channelInfo := strings.Split(name, "/")
				if len(channelInfo) > 1 {
					nodeInfo.NameSpace = channelInfo[0]
					name = channelInfo[1]
				} else {
					glog.V(2).Infof("For %s, %s edge not created as %s is not in namespace/name format", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind+"/"+name)
					continue
				}
			}
			if _, ok := ns.ByKindNamespaceName[destKind][nodeInfo.NameSpace][name]; ok {
				attachedToEdges = append(attachedToEdges, Edge{
					SourceUID: nodeInfo.UID,
					DestUID:   ns.ByKindNamespaceName[destKind][nodeInfo.NameSpace][name].UID,
					EdgeType:  nodeInfo.EdgeType,
				})
			} else {
				glog.V(2).Infof("For %s, %s edge not created as %s named %s not found", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind, nodeInfo.NameSpace+"/"+name)
			}
		}
		// If the destination node has property _ownerUID, create an edge between the pod and the destination's owner
		// Call the edgesByOwner recursively to create the uses edge
		if nextSrc, ok := ns.ByUID[nodeInfo.UID]; ok {
			if nextSrc.GetMetadata("OwnerUID") != "" {
				if nextSrcOwner, ok := ns.ByUID[nextSrc.GetMetadata("OwnerUID")]; ok {
					nodeInfo.UID = nextSrc.GetMetadata("OwnerUID")
					nodeInfo.Kind = nextSrcOwner.Properties["kind"].(string)
					nodeInfo.EdgeType = "uses"
					attachedToEdges = append(attachedToEdges, edgesByDestinationName(propSet, attachedToEdges, destKind, nodeInfo, ns)...)
				}
			}
		}
	}
	return attachedToEdges
}

// Function used to get edges to deployable and subscription
func edgesByDeployerSubscriber(nodeInfo NodeInfo, ns NodeStore) []Edge {
	ret := []Edge{}
	// Inner function used to connect to subscription and deployable
	edgesByDepSub := func(destNsName, destKind string) []Edge {
		depSubedges := []Edge{}

		if destNsName != "" && strings.Contains(destNsName, "/") {
			namespace := strings.Split(destNsName, "/")[0]
			name := strings.Split(destNsName, "/")[1]

			if _, ok := ns.ByKindNamespaceName[destKind][namespace][name]; ok {
				depSubedges = append(depSubedges, Edge{
					SourceUID: nodeInfo.UID,
					DestUID:   ns.ByKindNamespaceName[destKind][namespace][name].UID,
					EdgeType:  nodeInfo.EdgeType,
				})
			} else {
				glog.V(2).Infof("For %s, %s edge not created as %s named %s not found", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind, namespace+"/"+name)
			}
		} else {
			glog.V(2).Infof("For %s, %s edge not created as %s is not in namespace/name format", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destNsName)
		}
		return depSubedges
	}

	//Inner function to call edgesByDepSub for creating edges from node to hosting deployable/subscription - recursively calls with the owner's properties if the incoming node doesn't have them
	var findSub func(string) []Edge
	findSub = func(UID string) []Edge {
		subscription := ""
		deployable := ""
		if node, ok := ns.ByUID[UID]; ok {
			if subscription, ok = node.Properties["_app.ibm.com/hosting-subscription"].(string); ok && node.Properties["_app.ibm.com/hosting-subscription"] != "" {
				nodeInfo.EdgeType = "deployedBy"
				ret = append(ret, edgesByDepSub(subscription, "Subscription")...)
			}
			if deployable, ok = node.Properties["_app.ibm.com/hosting-deployable"].(string); ok && node.Properties["_app.ibm.com/hosting-deployable"] != "" {
				nodeInfo.EdgeType = "definedBy"
				ret = append(ret, edgesByDepSub(deployable, "Deployable")...)
			}
			// Recursively call the function with ownerUID, if the node doesn't have hosting deployable/subscription properties but has an owner reference.
			// This is mainly to create edges from pods to subscription/deployable, when the hosting deployable/subscription properties are not in pods, but present in deployments
			if subscription == "" && deployable == "" {
				if node.GetMetadata("OwnerUID") != "" {
					node = ns.ByUID[node.GetMetadata("OwnerUID")]
					ret = findSub(node.UID)
				}
			}
		}
		return ret
	}
	ret = findSub(nodeInfo.UID)
	return ret
}
