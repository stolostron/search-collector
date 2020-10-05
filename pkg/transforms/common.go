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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if resource.GetAnnotations()["apps.open-cluster-management.io/hosting-subscription"] != "" {
		ret["_hostingSubscription"] = resource.GetAnnotations()["apps.open-cluster-management.io/hosting-subscription"]
	}
	if resource.GetAnnotations()["apps.open-cluster-management.io/hosting-deployable"] != "" {
		ret["_hostingDeployable"] = resource.GetAnnotations()["apps.open-cluster-management.io/hosting-deployable"]
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
	//Adding OwnerReleaseName and Namespace for the resources that doesn't have ownerRef, but are deployed by a release
	if n.Metadata["OwnerUID"] == "" && resource.GetAnnotations()["meta.helm.sh/release-name"] != "" && resource.GetAnnotations()["meta.helm.sh/release-namespace"] != "" {
		n.Metadata["OwnerReleaseName"] = resource.GetAnnotations()["meta.helm.sh/release-name"]
		n.Metadata["OwnerReleaseNamespace"] = resource.GetAnnotations()["meta.helm.sh/release-namespace"]
	}
	return n
}

func addReleaseOwnerUID(node Node, ns NodeStore) {
	releaseNode, ok := ns.ByKindNamespaceName["HelmRelease"][node.GetMetadata("OwnerReleaseNamespace")][node.GetMetadata("OwnerReleaseName")]
	if ok { // If the HelmRelease node is in the list of current nodes
		node.Metadata["OwnerUID"] = releaseNode.UID
	} else {
		glog.V(3).Info("Release node not found with kind:HelmRelease, namespace: ", node.GetMetadata("OwnerReleaseNamespace"), " name: ", node.GetMetadata("OwnerReleaseName"))
	}
}

func CommonEdges(uid string, ns NodeStore) []Edge {
	ret := []Edge{}
	currNode := ns.ByUID[uid]
	namespace := ""
	kind := currNode.Properties["kind"].(string)
	if currNode.Properties["namespace"] != nil {
		namespace = currNode.Properties["namespace"].(string)
	} else { // If namespace property is not present, nodeTripleMap assigns namespace to be _NONE in reconciler (reconciler.go:47)
		namespace = "_NONE"
	}
	if currNode.Metadata["OwnerUID"] == "" && currNode.Metadata["OwnerReleaseName"] != "" && currNode.Metadata["OwnerReleaseNamespace"] != "" {
		addReleaseOwnerUID(currNode, ns) //add OwnerUID for resources deployed by HelmRelease, but doesn't have an associated ownerRef
		//mostly cluster-scoped resources like ClusterRole and ClusterRoleBinding
	}
	nodeInfo := NodeInfo{Name: currNode.Properties["name"].(string), NameSpace: namespace, UID: uid, EdgeType: "ownedBy", Kind: kind}

	//ownedBy edges
	if currNode.GetMetadata("OwnerUID") != "" {
		ret = append(ret, edgesByOwner(currNode.GetMetadata("OwnerUID"), ns, nodeInfo, []string{})...)
	}

	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}

// Prefixes the given UID with the cluster name from config and a /
func prefixedUID(uid apiTypes.UID) string {
	return strings.Join([]string{config.Cfg.ClusterName, string(uid)}, "/")
}

// func prefixedUIDStr(uid string) string {
// 	return strings.Join([]string{config.Cfg.ClusterName, string(uid)}, "/")
// }

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
func edgesByOwner(destUID string, ns NodeStore, nodeInfo NodeInfo, seenDests []string) []Edge {
	ret := []Edge{}
	for _, value := range seenDests {
		if value == destUID {
			return ret
		}
	}
	if destUID != "" {
		//Lookup by UID to see if the owner Node exists
		if dest, ok := ns.ByUID[destUID]; ok {
			if nodeInfo.UID != destUID { //avoid connecting node to itself
				ret = append(ret, Edge{
					SourceUID:  nodeInfo.UID,
					DestUID:    destUID,
					EdgeType:   nodeInfo.EdgeType,
					SourceKind: nodeInfo.Kind,
					DestKind:   dest.Properties["kind"].(string),
				})
				seenDests = append(seenDests, destUID)    //add destUID to processed/seen destinations
				if dest.GetMetadata("ReleaseUID") != "" { // If owner included/owned by a release...
					if _, ok := ns.ByUID[dest.GetMetadata("ReleaseUID")]; ok { // ...make sure the release exists...
						if nodeInfo.UID != dest.GetMetadata("ReleaseUID") { //avoid connecting node to itself
							ret = append(ret, Edge{ // ... then add edge from source to release
								SourceUID:  nodeInfo.UID,
								DestUID:    dest.GetMetadata("ReleaseUID"),
								EdgeType:   "ownedBy",
								SourceKind: nodeInfo.Kind,
								DestKind:   dest.Properties["kind"].(string),
							})
						}
					}
				}

				// If the destination node has property _ownerUID, create an edge between the pod and the destination's owner
				// Call the edgesByOwner recursively to create the ownedBy edge
				if dest.GetMetadata("OwnerUID") != "" {
					ret = append(ret, edgesByOwner(dest.GetMetadata("OwnerUID"), ns, nodeInfo, seenDests)...)
				}
			}
		} else {
			glog.V(4).Infof("For %s, %s, %s edge not created: ownerUID %s not found", nodeInfo.Kind, nodeInfo.NameSpace+"/"+nodeInfo.Name, nodeInfo.EdgeType, destUID)
		}
	}
	return ret
}

// Function used to get all edges for a specific destKind - the propSet are maps of resource names, nodeInfo has additional info about the node and nodestore has all the current nodes
func edgesByDestinationName(propSet map[string]struct{}, destKind string, nodeInfo NodeInfo, ns NodeStore, seenDests []string) []Edge {
	ret := []Edge{}
	for _, value := range seenDests {
		//Checking against nodeInfo.UID - it gets updated every time edgesByDestinationName is called
		if value == nodeInfo.UID {
			return ret
		}
	}

	if len(propSet) > 0 {
		for name := range propSet {
			// For channels/subscriptions/deployables/applications, get the namespace and name from each string, if present. Else, assume it is in the node's namespace
			if destKind == "Channel" || destKind == "Deployable" || destKind == "Subscription" || destKind == "Application" {
				destKindInfo := strings.Split(name, "/")
				if len(destKindInfo) == 2 {
					nodeInfo.NameSpace = destKindInfo[0]
					name = destKindInfo[1]
				} else if len(destKindInfo) == 1 {
					name = destKindInfo[0]
				} else {
					glog.V(4).Infof("For %s, %s edge not created as %s is not in namespace/name format", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind+"/"+name)
					continue
				}
			}
			if destNode, ok := ns.ByKindNamespaceName[destKind][nodeInfo.NameSpace][name]; ok {
				if nodeInfo.UID != destNode.UID { //avoid connecting node to itself
					ret = append(ret, Edge{
						SourceUID:  nodeInfo.UID,
						DestUID:    destNode.UID,
						EdgeType:   nodeInfo.EdgeType,
						SourceKind: nodeInfo.Kind,
						DestKind:   destKind,
					})
					//Add all the applications connected to a subscription in the Subscription  node's metadata - this metadata will be used to connect other nodes to Application
					if destKind == "Subscription" && nodeInfo.Kind == "Application" {
						if destNode.Metadata["_hostingApplication"] != "" {
							currAppInfo := nodeInfo.NameSpace + "/" + nodeInfo.Name
							if !strings.Contains(destNode.Metadata["_hostingApplication"], currAppInfo) {
								destNode.Metadata["_hostingApplication"] = destNode.Metadata["_hostingApplication"] + "," + nodeInfo.NameSpace + "/" + nodeInfo.Name
							}
						} else {
							destNode.Metadata["_hostingApplication"] = nodeInfo.NameSpace + "/" + nodeInfo.Name
						}
					} else if destKind == "Subscription" && nodeInfo.Kind != "Application" { //Connect incoming node to all applications in the Subscription node's metadata
						ret = append(ret, edgesToApplication(nodeInfo, ns, destNode.UID, false)...)
					} else if nodeInfo.Kind == "Subscription" && (destKind == "Deployable" || destKind == "PlacementRule" || destKind == "Channel") { // Build edges between all applications connected to the subscription (using metadata _hostingApplication) to deployables, placementrule and channel
						subUID := nodeInfo.UID
						nodeInfoDestApp := NodeInfo{UID: destNode.UID, Name: name, NameSpace: nodeInfo.NameSpace, Kind: destKind, EdgeType: "contains"}
						ret = append(ret, edgesToApplication(nodeInfoDestApp, ns, subUID, true)...)
					}
				}
			} else {
				glog.V(4).Infof("For %s, %s edge not created as %s named %s not found", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind, nodeInfo.NameSpace+"/"+name)
			}
		}
		seenDests = append(seenDests, nodeInfo.UID) //add nodeInfo UID to processed/seen nodes

		// If the destination node has property _ownerUID, create an edge between the pod and the destination's owner
		// Call the edgesByOwner recursively to create the uses edge
		if nodeInfo.Kind != "Deployable" { //Adding this edge case to avoid duplicating edges between subscription to placementrules and applications
			//deployable's owner will be subscription - this edge is already covered in subscription
			if nextSrc, ok := ns.ByUID[nodeInfo.UID]; ok {
				if nextSrc.GetMetadata("OwnerUID") != "" {
					if nextSrcOwner, ok := ns.ByUID[nextSrc.GetMetadata("OwnerUID")]; ok {
						nodeInfo.UID = nextSrc.GetMetadata("OwnerUID")
						nodeInfo.Kind = nextSrcOwner.Properties["kind"].(string)
						nodeInfo.EdgeType = "uses"
						ret = append(ret, edgesByDestinationName(propSet, destKind, nodeInfo, ns, seenDests)...)
					}
				}
			}
		}
	}
	return ret
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

			if dest, ok := ns.ByKindNamespaceName[destKind][namespace][name]; ok {
				if nodeInfo.UID != dest.UID { //avoid connecting node to itself
					depSubedges = append(depSubedges, Edge{
						SourceUID:  nodeInfo.UID,
						DestUID:    dest.UID,
						EdgeType:   nodeInfo.EdgeType,
						SourceKind: nodeInfo.Kind,
						DestKind:   dest.Properties["kind"].(string),
					})
					//Connect incoming node to all applications in the Subscription node's metadata
					if destKind == "Subscription" && nodeInfo.Kind != "Application" {
						depSubedges = append(depSubedges, edgesToApplication(nodeInfo, ns, dest.UID, false)...)
					} else if nodeInfo.Kind == "Subscription" && destKind == "Deployable" { // Build edges between all applications connected to the subscription (using metadata _hostingApplication) to the hosting-deployable
						subUID := nodeInfo.UID
						nodeInfoDestApp := NodeInfo{UID: dest.UID, Name: name, NameSpace: namespace, Kind: destKind, EdgeType: "contains"}
						depSubedges = append(depSubedges, edgesToApplication(nodeInfoDestApp, ns, subUID, true)...)
					}
				}
			} else {
				glog.V(4).Infof("For %s, %s edge not created as %s named %s not found", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind, namespace+"/"+name)
			}
		} else {
			glog.V(4).Infof("For %s, %s edge not created as %s is not in namespace/name format", nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destNsName)
		}
		return depSubedges
	}

	//Inner function to call edgesByDepSub for creating edges from node to hosting deployable/subscription - recursively calls with the owner's properties if the incoming node doesn't have them
	var findSub func(string) []Edge
	var seenDests []string
	findSub = func(UID string) []Edge {
		//Checking against UID - it gets updated every time findSub is called
		for _, value := range seenDests {
			if value == UID {
				return ret
			}
		}
		subscription := ""
		deployable := ""
		if node, ok := ns.ByUID[UID]; ok {
			if subscription, ok = node.Properties["_hostingSubscription"].(string); ok && node.Properties["_hostingSubscription"] != "" {
				nodeInfo.EdgeType = "deployedBy"
				ret = append(ret, edgesByDepSub(subscription, "Subscription")...)
			}
			if deployable, ok = node.Properties["_hostingDeployable"].(string); ok && node.Properties["_hostingDeployable"] != "" {
				nodeInfo.EdgeType = "definedBy"
				ret = append(ret, edgesByDepSub(deployable, "Deployable")...)
			}
			seenDests = append(seenDests, UID) //add UID to processed/seen destinations

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

//Build edges from the source node in nodeInfo to all applications/channels in the subscription's metadata. UID is the subscription node's UID. Connect to only application if the onlyApplication parameter is true
func edgesToApplication(nodeInfo NodeInfo, ns NodeStore, UID string, onlyApplication bool) []Edge {
	ret := []Edge{}
	// Connect all applications connected to the subscription (using metadata _hostingApplication)
	subNode := ns.ByUID[UID]
	if subNode.GetMetadata("_hostingApplication") != "" {
		applicationMap := make(map[string]struct{})
		for _, app := range strings.Split(subNode.GetMetadata("_hostingApplication"), ",") {
			applicationMap[app] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(applicationMap, "Application", nodeInfo, ns, []string{})...)
	}
	if !onlyApplication {
		if subNode.GetMetadata("_channels") != "" {
			channelMap := make(map[string]struct{})
			for _, channel := range strings.Split(subNode.GetMetadata("_channels"), ",") {
				channelMap[channel] = struct{}{}
			}
			ret = append(ret, edgesByDestinationName(channelMap, "Channel", nodeInfo, ns, []string{})...)
		}
	}
	return ret
}

// SliceDiff returns the elements in bigSlice that aren't in smallSlice
func SliceDiff(bigSlice, smallSlice []string) []string {
	smallMap := make(map[string]struct{}, len(smallSlice))
	for _, elem := range smallSlice {
		smallMap[elem] = struct{}{}
	}

	var diff []string

	for _, elem := range bigSlice {
		if _, ok := smallMap[elem]; !ok {
			diff = append(diff, elem)
		}
	}
	return diff
}

func apiGroupVersion(typeMeta v1.TypeMeta, node *Node) {
	node.Properties["kind"] = typeMeta.Kind
	apiVersion := strings.Split(typeMeta.APIVersion, "/")
	if len(apiVersion) == 2 {
		node.Properties["apigroup"] = apiVersion[0]
		node.Properties["apiversion"] = apiVersion[1]
	} else {
		node.Properties["apiversion"] = apiVersion[0]
	}
}

// Copy hosting Subscription/Deployable properties from the sourceNode to the destination
func copyhostingSubProperties(srcUID string, destUID string, ns NodeStore) {
	srcNode, srcFound := ns.ByUID[srcUID]
	destNode, destFound := ns.ByUID[destUID]

	subscription := ""
	deployable := ""
	ok := false
	// Copy the properties to the destination - this makes it easy to connect them back to the subscription/application
	if srcFound && destFound {
		if subscription, ok = srcNode.Properties["_hostingSubscription"].(string); ok && srcNode.Properties["_hostingSubscription"] != "" {
			if destNode.Properties["_hostingSubscription"] != subscription {
				destNode.Properties["_hostingSubscription"] = subscription
			}
		}
		if deployable, ok = srcNode.Properties["_hostingDeployable"].(string); ok && srcNode.Properties["_hostingDeployable"] != "" {
			if destNode.Properties["_hostingDeployable"] != deployable {
				destNode.Properties["_hostingDeployable"] = deployable
			}
		}

		// If both properties are not there on source, check if it is on it's owner - This will be the case if the pod doesn't have the properties but the deployment has
		if subscription == "" && deployable == "" {
			if srcNode.GetMetadata("OwnerUID") != "" {
				node := ns.ByUID[srcNode.GetMetadata("OwnerUID")]
				copyhostingSubProperties(node.UID, destUID, ns)
			}
		}
	}
}

//Given UID returns if there is any subscription attached to iself or its parents
// func getSubscriptionByUID(srcUID string, ns NodeStore) string {
// 	subscriptionUID := ""
// 	srcNode, ok := ns.ByUID[srcUID]
// 	if ok {
// 		if subscription, ok := srcNode.Properties["_hostingSubscription"].(string); ok && srcNode.Properties["_hostingSubscription"] != "" {
// 			nsSub := strings.Split(subscription, "/")
// 			if len(nsSub) == 2 {
// 				nameSpace := nsSub[0]
// 				name := nsSub[1]
// 				subscriptionUID = ns.ByKindNamespaceName["Subscription"][nameSpace][name].UID
// 				return subscriptionUID
// 			}
// 		} else if srcNode.GetMetadata("OwnerUID") != "" {
// 			subscriptionUID = getSubscriptionByUID(srcNode.GetMetadata("OwnerUID"), ns)
// 		}
// 	}
// 	return subscriptionUID
// }
