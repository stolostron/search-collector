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
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
)

// An object given to the Edge Building methods in the transforms package.
// Contains representations of the Node list that are useful for them to efficiently find the nodes that they need.
type NodeStore struct {
	ByUID               map[string]Node
	ByKindNamespaceName map[string]map[string]map[string]Node
}

// commonAnnotations returns the annotations with values <= 64 characters. It also removes the
// last-applied-configuration annotation regardless of length.
func commonAnnotations(object v1.Object) map[string]string {
	// If CollectAnnotations is not true, then only collect annotations for allow listed resources.
	if !config.Cfg.CollectAnnotations {
		objKind, ok := object.(schema.ObjectKind)
		if !ok {
			return nil
		}

		switch objKind.GroupVersionKind().Group {
		case POLICY_OPEN_CLUSTER_MANAGEMENT_IO:
		// Add annotations for severity
		case "constraints.gatekeeper.sh":
		case "mutations.gatekeeper.sh":
		default:
			transformConfig, found := getTransformConfig(objKind.GroupVersionKind().Group, objKind.GroupVersionKind().Kind)
			if found && transformConfig.extractAnnotations {
				break
			}
			return nil
		}
	}

	annotations := object.GetAnnotations()

	// This annotation is large and useless
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	for key, val := range annotations {
		if utf8.RuneCountInString(val) > 64 {
			delete(annotations, key)
		}
	}

	return annotations
}

// Extracts the common properties from a k8s resource of any type and returns a map ready to be put in a Node
func commonProperties(resource v1.Object) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["name"] = resource.GetName()
	ret["created"] = resource.GetCreationTimestamp().UTC().Format(time.RFC3339)
	if config.Cfg.DeployedInHub {
		ret["_hubClusterResource"] = true
	}

	labels := resource.GetLabels()
	if labels != nil {
		ret["label"] = labels
	}

	annotations := commonAnnotations(resource)

	if annotations != nil {
		ret["annotation"] = annotations
	}

	namespace := resource.GetNamespace()
	if namespace != "" {
		ret["namespace"] = namespace
	}

	hostingSubscription := resource.GetAnnotations()["apps.open-cluster-management.io/hosting-subscription"]
	if hostingSubscription != "" {
		ret["_hostingSubscription"] = hostingSubscription
	}

	hostingDeployable := resource.GetAnnotations()["apps.open-cluster-management.io/hosting-deployable"]
	if hostingDeployable != "" {
		ret["_hostingDeployable"] = hostingDeployable
	}
	return ret
}

// Transforms a resource of unknown type by simply pulling out the common properties.
func transformCommon(resource v1.Object) Node {
	n := Node{
		UID:        prefixedUID(resource.GetUID()),
		Properties: commonProperties(resource),
		Metadata:   make(map[string]any),
	}

	// When a resource is mutated by Gatekeeper, add this annotation
	mutation, ok := resource.GetAnnotations()["gatekeeper.sh/mutations"]
	if ok {
		n.Metadata["gatekeeper.sh/mutations"] = mutation
	}

	n.Metadata["OwnerUID"] = ownerRefUID(resource.GetOwnerReferences())
	// Adding OwnerReleaseName and Namespace to resources that doesn't have ownerRef but are deployed by a release.
	if n.Metadata["OwnerUID"] == "" && resource.GetAnnotations()["meta.helm.sh/release-name"] != "" &&
		resource.GetAnnotations()["meta.helm.sh/release-namespace"] != "" {
		n.Metadata["OwnerReleaseName"] = resource.GetAnnotations()["meta.helm.sh/release-name"]
		n.Metadata["OwnerReleaseNamespace"] = resource.GetAnnotations()["meta.helm.sh/release-namespace"]
	}

	return n
}

func addReleaseOwnerUID(node Node, ns NodeStore) {
	ownerNamespace := node.GetMetadata("OwnerReleaseNamespace")
	ownerName := node.GetMetadata("OwnerReleaseName")

	// If the HelmRelease node is in the list of current nodes
	if releaseNode, ok := ns.ByKindNamespaceName["HelmRelease"][ownerNamespace][ownerName]; ok {
		node.Metadata["OwnerUID"] = releaseNode.UID
	} else {
		klog.V(3).Infof("HelmRelease node not found for namespace: %s name: %s", ownerNamespace, ownerName)
	}
}

func CommonEdges(uid string, ns NodeStore) []Edge {
	ret := []Edge{}
	currNode := ns.ByUID[uid]
	namespace := ""
	kind := currNode.Properties["kind"].(string)
	if currNode.Properties["namespace"] != nil {
		namespace = currNode.Properties["namespace"].(string)
	} else {
		// If namespace property is not present, nodeTripleMap assigns namespace to be
		// _NONE in reconciler (reconciler.go:47)
		namespace = "_NONE"
	}
	if currNode.Metadata["OwnerUID"] == "" && currNode.Metadata["OwnerReleaseName"] != "" &&
		currNode.Metadata["OwnerReleaseNamespace"] != "" {
		// add OwnerUID for resources deployed by HelmRelease, but doesn't have an associated ownerRef
		// mostly cluster-scoped resources like ClusterRole and ClusterRoleBinding
		addReleaseOwnerUID(currNode, ns)
	}
	nodeInfo := NodeInfo{
		Name:      currNode.Properties["name"].(string),
		NameSpace: namespace,
		UID:       uid,
		EdgeType:  "ownedBy",
		Kind:      kind,
	}

	// ownedBy edges
	if currNode.GetMetadata("OwnerUID") != "" {
		ret = append(ret, edgesByOwner(currNode.GetMetadata("OwnerUID"), ns, nodeInfo, []string{})...)
	}

	// deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)

	ret = edgesByKyverno(ret, currNode, ns)

	ret = edgesByGatekeeperMutation(ret, currNode, ns)

	ret = edgesByDefaultTransformConfig(ret, currNode, ns)

	return ret
}

// Function to create a configured edge between configured resources
func edgesByDefaultTransformConfig(ret []Edge, currNode Node, ns NodeStore) []Edge {
	var kind, apiGroup, namespace string
	kind = currNode.Properties["kind"].(string)
	if n, ok := currNode.Properties["namespace"]; ok {
		namespace = n.(string)
	}
	if g, ok := currNode.Properties["apigroup"]; ok {
		apiGroup = g.(string)
	}

	transformConfig, found := getTransformConfig(apiGroup, kind)

	if found {
		for _, e := range transformConfig.edges {
			var val interface{}
			if v, ok := currNode.Metadata[e.Name]; ok {
				val = v
			} else if v, ok = currNode.Properties[e.Name]; ok {
				val = v
			}
			switch v := val.(type) {
			case string:
				n, ok := ns.ByKindNamespaceName[e.ToKind][namespace][v]
				if !ok {
					continue
				}
				ret = append(ret, Edge{
					SourceKind: kind,
					SourceUID:  currNode.UID,
					EdgeType:   e.Type,
					DestKind:   n.Properties["kind"].(string),
					DestUID:    n.UID,
				})
			case []interface{}:
				for _, item := range v {
					n, ok := ns.ByKindNamespaceName[e.ToKind][namespace][item.(string)]
					if !ok {
						continue
					}
					ret = append(ret, Edge{
						SourceKind: kind,
						SourceUID:  currNode.UID,
						EdgeType:   e.Type,
						DestKind:   n.Properties["kind"].(string),
						DestUID:    n.UID,
					})
				}
			}
		}
	}

	return ret
}

// Function to create an edge linking any resource with a Kyverno Policy or ClusterPolicy that generates the resource.
func edgesByKyverno(ret []Edge, currNode Node, ns NodeStore) []Edge {
	labels, ok := currNode.Properties["label"].(map[string]string)
	if !ok {
		return ret
	}

	if labels["app.kubernetes.io/managed-by"] != "kyverno" || labels["generate.kyverno.io/policy-name"] == "" {
		return ret
	}

	// For resources created by kyverno
	policyNamespace := labels["generate.kyverno.io/policy-namespace"]
	policyName := labels["generate.kyverno.io/policy-name"]
	// Kyverno Policy
	policyKind := "Policy"

	if policyNamespace == "" {
		// Kyverno ClusterPolicy
		policyKind = "ClusterPolicy"
		policyNamespace = "_NONE"
	}

	policyNode, ok := ns.ByKindNamespaceName[policyKind][policyNamespace][policyName]
	if !ok {
		return ret
	}

	// Prevent from policy.policy.open-cluster-management.io
	if policyNode.Properties["apigroup"] != "kyverno.io" {
		return ret
	}

	ret = append(ret, Edge{
		SourceKind: currNode.Properties["kind"].(string),
		SourceUID:  currNode.UID,
		EdgeType:   "generatedBy",
		DestUID:    policyNode.UID,
		DestKind:   policyNode.Properties["kind"].(string),
	})

	return ret
}

// Function to create an edge linking a resource to Gatekeeper mutations (e.g., Assign, AssignImage) that modify the resource.
func edgesByGatekeeperMutation(ret []Edge, currNode Node, ns NodeStore) []Edge {
	mutationEntries := currNode.GetMetadata("gatekeeper.sh/mutations")
	if mutationEntries == "" {
		return ret
	}

	for _, kindNsName := range strings.Split(mutationEntries, ", ") {
		parts := strings.Split(kindNsName, "/")
		if len(parts) != 3 {
			continue // Skip invalid entries that don't follow the "kind/namespace/name" format
		}

		mutationNs := parts[1]
		if parts[1] == "" {
			mutationNs = "_NONE"
		}

		// Extract the mutation name (ignoring any suffix after ':').
		mutationName := strings.Split(parts[2], ":")[0]
		mutationNode, ok := ns.ByKindNamespaceName[parts[0]][mutationNs][mutationName]
		if !ok {
			continue
		}

		ret = append(ret, Edge{
			SourceKind: currNode.Properties["kind"].(string),
			SourceUID:  currNode.UID,
			EdgeType:   "mutatedBy",
			DestUID:    mutationNode.UID,
			DestKind:   mutationNode.Properties["kind"].(string),
		})
	}

	return ret
}

// Prefixes the given UID with the cluster name from config and a /
func prefixedUID(uid apiTypes.UID) string {
	return strings.Join([]string{config.Cfg.ClusterName, string(uid)}, "/")
}

// Prefixes the given UID with the cluster name from config and a /
func ownerRefUID(ownerReferences []v1.OwnerReference) string {
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
// If the pod is owned by a replicaset which in turn is owned by a deployment, the function will be recursively called
// to create edges between pod->replicaset and pod->deployment
func edgesByOwner(destUID string, ns NodeStore, nodeInfo NodeInfo, seenDests []string) []Edge {
	ret := []Edge{}
	for _, value := range seenDests {
		if value == destUID {
			return ret
		}
	}
	if destUID != "" {
		// Lookup by UID to see if the owner Node exists
		if dest, ok := ns.ByUID[destUID]; ok {
			if nodeInfo.UID != destUID { // avoid connecting node to itself
				ret = append(ret, Edge{
					SourceUID:  nodeInfo.UID,
					DestUID:    destUID,
					EdgeType:   nodeInfo.EdgeType,
					SourceKind: nodeInfo.Kind,
					DestKind:   dest.Properties["kind"].(string),
				})
				seenDests = append(seenDests, destUID)    // add destUID to processed/seen destinations
				if dest.GetMetadata("ReleaseUID") != "" { // If owner included/owned by a release...
					if _, ok := ns.ByUID[dest.GetMetadata("ReleaseUID")]; ok && // ...make sure the release exists...
						nodeInfo.UID != dest.GetMetadata("ReleaseUID") { // avoid connecting node to itself
						ret = append(ret, Edge{ // ... then add edge from source to release
							SourceUID:  nodeInfo.UID,
							DestUID:    dest.GetMetadata("ReleaseUID"),
							EdgeType:   "ownedBy",
							SourceKind: nodeInfo.Kind,
							DestKind:   dest.Properties["kind"].(string),
						})
					}
				}

				// If the destination node has property _ownerUID, create an edge between the pod and the dest owner
				// Call the edgesByOwner recursively to create the ownedBy edge
				if dest.GetMetadata("OwnerUID") != "" {
					ret = append(ret, edgesByOwner(dest.GetMetadata("OwnerUID"), ns, nodeInfo, seenDests)...)
				}
			}
		} else {
			klog.V(4).Infof("For %s, %s, %s edge not created: ownerUID %s not found",
				nodeInfo.Kind, nodeInfo.NameSpace+"/"+nodeInfo.Name, nodeInfo.EdgeType, destUID)
		}
	}
	return ret
}

// Function used to get all edges for a specific destKind - the propSet are maps of resource names,
// nodeInfo has additional info about the node and nodestore has all the current nodes
func edgesByDestinationName(
	propSet map[string]struct{},
	destKind string,
	nodeInfo NodeInfo,
	ns NodeStore,
	seenDests []string,
) []Edge {
	ret := []Edge{}
	for _, value := range seenDests {
		// Checking against nodeInfo.UID - it gets updated every time edgesByDestinationName is called
		if value == nodeInfo.UID {
			return ret
		}
	}

	if len(propSet) > 0 {
		for name := range propSet {
			// For channels/subscriptions/deployables/applications, get the namespace and name from each string,
			// if present. Else, assume it is in the node's namespace
			if destKind == "Channel" || destKind == "Deployable" || destKind == "Subscription" ||
				destKind == "Application" {
				destKindInfo := strings.Split(name, "/")
				if len(destKindInfo) == 2 {
					nodeInfo.NameSpace = destKindInfo[0]
					name = destKindInfo[1]
				} else if len(destKindInfo) == 1 {
					name = destKindInfo[0]
				} else {
					klog.V(4).Infof("For %s, %s edge not created as %s is not in namespace/name format",
						nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind+"/"+name)
					continue
				}
			}
			if destNode, ok := ns.ByKindNamespaceName[destKind][nodeInfo.NameSpace][name]; ok {
				if nodeInfo.UID != destNode.UID { // avoid connecting node to itself
					ret = append(ret, Edge{
						SourceUID:  nodeInfo.UID,
						DestUID:    destNode.UID,
						EdgeType:   nodeInfo.EdgeType,
						SourceKind: nodeInfo.Kind,
						DestKind:   destKind,
					})
					// Add all the applications connected to a subscription in the Subscription node's metadata -
					// this metadata will be used to connect other nodes to Application
					if destKind == "Subscription" && nodeInfo.Kind == "Application" {
						if destNode.GetMetadata("_hostingApplication") != "" {
							currAppInfo := nodeInfo.NameSpace + "/" + nodeInfo.Name
							if !strings.Contains(destNode.GetMetadata("_hostingApplication"), currAppInfo) {
								destNode.Metadata["_hostingApplication"] = destNode.GetMetadata("_hostingApplication") +
									"," + nodeInfo.NameSpace + "/" + nodeInfo.Name
							}
						} else {
							destNode.Metadata["_hostingApplication"] = nodeInfo.NameSpace + "/" + nodeInfo.Name
						}
					} else if destKind == "Subscription" && nodeInfo.Kind != "Application" {
						// Connect incoming node to all applications in the Subscription node's metadata
						ret = append(ret, edgesToApplication(nodeInfo, ns, destNode.UID, false)...)
					} else if nodeInfo.Kind == "Subscription" && (destKind == "Deployable" ||
						destKind == "PlacementRule" || destKind == "Channel") {
						// Build edges between all applications connected to the subscription (using metadata
						// _hostingApplication) to deployables, placementrule and channel
						subUID := nodeInfo.UID
						nodeInfoDestApp := NodeInfo{
							UID:       destNode.UID,
							Name:      name,
							NameSpace: nodeInfo.NameSpace,
							Kind:      destKind,
							EdgeType:  "contains",
						}
						ret = append(ret, edgesToApplication(nodeInfoDestApp, ns, subUID, true)...)
					}
				}
			} else {
				klog.V(4).Infof("For %s, %s edge not created as %s named %s not found",
					nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name,
					nodeInfo.EdgeType, destKind, nodeInfo.NameSpace+"/"+name)
			}
		}
		seenDests = append(seenDests, nodeInfo.UID) // add nodeInfo UID to processed/seen nodes

		// If the destination node has property _ownerUID, create an edge between the pod and the destination's owner
		// Call the edgesByOwner recursively to create the uses edge
		if nodeInfo.Kind != "Deployable" {
			// Adding this edge case to avoid duplicating edges between subscription to placementrules and applications
			// deployable's owner will be subscription - this edge is already covered in subscription
			if nextSrc, ok := ns.ByUID[nodeInfo.UID]; ok && nextSrc.GetMetadata("OwnerUID") != "" {
				if nextSrcOwner, ok := ns.ByUID[nextSrc.GetMetadata("OwnerUID")]; ok {
					nodeInfo.UID = nextSrc.GetMetadata("OwnerUID")
					nodeInfo.Kind = nextSrcOwner.Properties["kind"].(string)
					nodeInfo.EdgeType = "uses"
					ret = append(ret, edgesByDestinationName(propSet, destKind, nodeInfo, ns, seenDests)...)
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
				if nodeInfo.UID != dest.UID { // avoid connecting node to itself
					depSubedges = append(depSubedges, Edge{
						SourceUID:  nodeInfo.UID,
						DestUID:    dest.UID,
						EdgeType:   nodeInfo.EdgeType,
						SourceKind: nodeInfo.Kind,
						DestKind:   dest.Properties["kind"].(string),
					})
					// Connect incoming node to all applications in the Subscription node's metadata
					if destKind == "Subscription" && nodeInfo.Kind != "Application" {
						depSubedges = append(depSubedges, edgesToApplication(nodeInfo, ns, dest.UID, false)...)
					} else if nodeInfo.Kind == "Subscription" && destKind == "Deployable" {
						// Build edges between all applications connected to the subscription
						// (using metadata _hostingApplication) to the hosting-deployable
						subUID := nodeInfo.UID
						nodeInfoDestApp := NodeInfo{
							UID:       dest.UID,
							Name:      name,
							NameSpace: namespace,
							Kind:      destKind,
							EdgeType:  "contains",
						}
						depSubedges = append(depSubedges, edgesToApplication(nodeInfoDestApp, ns, subUID, true)...)
					}
				}
			} else {
				klog.V(4).Infof("For %s, %s edge not created as %s named %s not found",
					nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destKind, namespace+"/"+name)
			}
		} else {
			klog.V(4).Infof("For %s, %s edge not created as %s is not in namespace/name format",
				nodeInfo.NameSpace+"/"+nodeInfo.Kind+"/"+nodeInfo.Name, nodeInfo.EdgeType, destNsName)
		}
		return depSubedges
	}

	// Inner function to call edgesByDepSub for creating edges from node to hosting deployable/subscription -
	// recursively calls with the owner's properties if the incoming node doesn't have them
	var findSub func(string) []Edge
	var seenDests []string
	findSub = func(UID string) []Edge {
		// Checking against UID - it gets updated every time findSub is called
		for _, value := range seenDests {
			if value == UID {
				return ret
			}
		}
		subscription := ""
		deployable := ""
		if node, ok := ns.ByUID[UID]; ok {
			if subscription, ok = node.Properties["_hostingSubscription"].(string); ok &&
				node.Properties["_hostingSubscription"] != "" {
				nodeInfo.EdgeType = "deployedBy"
				ret = append(ret, edgesByDepSub(subscription, "Subscription")...)
			}
			if deployable, ok = node.Properties["_hostingDeployable"].(string); ok &&
				node.Properties["_hostingDeployable"] != "" {
				nodeInfo.EdgeType = "definedBy"
				ret = append(ret, edgesByDepSub(deployable, "Deployable")...)
			}
			seenDests = append(seenDests, UID) // add UID to processed/seen destinations

			// Recursively call the function with ownerUID, if the node doesn't have hosting deployable/subscription
			// properties but has an owner reference.
			// This is mainly to create edges from pods to subscription/deployable, when the hosting
			// deployable/subscription properties are not in pods, but present in deployments
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

// Build edges from the source node in nodeInfo to all applications/channels in the subscription's metadata.
// UID is the subscription node's UID. Connect to only application if the onlyApplication parameter is true
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

// relatedObject stores identifying information for a kubernetes resource.
// When marshalled to json, the struct names are small to reduce total size.
type relatedObject struct {
	Group     string   `json:"g,omitempty"`
	Version   string   `json:"v,omitempty"`
	Kind      string   `json:"k,omitempty"`
	Namespace string   `json:"ns,omitempty"`
	Name      string   `json:"n,omitempty"`
	EdgeType  EdgeType `json:"-"` // omitted from serialization
}

func (r relatedObject) String() string {
	jsonBytes, _ := json.Marshal(r)
	return string(jsonBytes)
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
		if subscription, ok = srcNode.Properties["_hostingSubscription"].(string); ok &&
			srcNode.Properties["_hostingSubscription"] != "" {
			if destNode.Properties["_hostingSubscription"] != subscription {
				destNode.Properties["_hostingSubscription"] = subscription
			}
		}
		if deployable, ok = srcNode.Properties["_hostingDeployable"].(string); ok &&
			srcNode.Properties["_hostingDeployable"] != "" {
			if destNode.Properties["_hostingDeployable"] != deployable {
				destNode.Properties["_hostingDeployable"] = deployable
			}
		}

		// If both properties are not there on source, check if it is on it's owner - This will be the case if
		// the pod doesn't have the properties but the deployment has
		if subscription == "" && deployable == "" {
			if srcNode.GetMetadata("OwnerUID") != "" {
				node := ns.ByUID[srcNode.GetMetadata("OwnerUID")]
				copyhostingSubProperties(node.UID, destUID, ns)
			}
		}
	}
}

func applyDefaultTransformConfig(node Node, r *unstructured.Unstructured, additionalColumns ...ExtractProperty) Node {
	group := r.GroupVersionKind().Group
	kind := r.GetKind()
	// Check if a transform config exists for this resource and extract the additional properties.
	transformConfig, found := getTransformConfig(group, kind)

	if config.Cfg.CollectStatusConditions || (found && transformConfig.extractConditions) {
		conditionsMap := commonStatusConditions(kind, group, r)
		if len(conditionsMap) > 0 {
			node.Properties["condition"] = conditionsMap
		}
	}

	// Currently, only pull in the additionalPrinterColumns listed in the CRD if it's a Gatekeeper
	// constraint or globally enabled.
	if !found && (config.Cfg.CollectCRDPrinterColumns || group == "constraints.gatekeeper.sh") {
		transformConfig = ResourceConfig{properties: additionalColumns}
	} else if !found {
		return node
	}

	for _, prop := range transformConfig.properties {
		// Skip if property has matchLabel condition and node doesn't contain matching label
		if prop.matchLabel != "" && node.Properties["label"] != nil {
			if _, ok := node.Properties["label"].(map[string]string)[prop.matchLabel]; !ok {
				continue
			}
		}
		// Skip properties that are already set. This could happen if additionalPrinterColumns
		// is overriding a generic property.
		if !prop.metadataOnly {
			if _, ok := node.Properties[prop.Name]; ok {
				continue
			}
		}

		// Skip additionalPrinterColumns that should be ignored.
		if !found && defaultTransformIgnoredFields[prop.Name] {
			continue
		}

		jp := jsonpath.New(prop.Name)
		parseErr := jp.Parse(prop.JSONPath)
		if parseErr != nil {
			klog.Errorf("Error parsing jsonpath [%s] for [%s.%s] prop: [%s]. Reason: %v",
				prop.JSONPath, kind, group, prop.Name, parseErr)
			continue
		}

		result, err := jp.FindResults(r.Object)
		if err != nil {
			// This error isn't always indicative of a problem, for example, when the object is created, it
			// won't have a status yet, so the jsonpath returns an error until controller adds the status.
			klog.V(1).Infof("Unable to extract prop [%s] from [%s.%s] Name: [%s]. Reason: %v",
				prop.Name, kind, group, r.GetName(), err)
			continue
		}

		if len(result) > 0 && len(result[0]) > 0 {
			if prop.DataType == DataTypeSlice {
				var slice []interface{}
				for _, v := range result[0] {
					slice = append(slice, v.Interface())
				}
				if prop.metadataOnly {
					node.Metadata[prop.Name] = slice
				} else {
					node.Properties[prop.Name] = slice
				}
				continue
			}
			val := result[0][0].Interface()

			if knownStringArrays[prop.Name] {
				if _, ok := val.([]string); !ok {
					klog.V(1).Infof("Ignoring the property [%s] from [%s.%s] Name: [%s]. Reason: not a string slice",
						prop.Name, kind, group, r.GetName())
					continue
				}
			}

			if prop.metadataOnly {
				strVal, ok := val.(string)

				if !ok {
					klog.V(1).Infof(
						"Unable to extract metadata prop [%s] from [%s.%s] Name: [%s] since it's not a string: %v",
						prop.Name, kind, group, r.GetName(), val,
					)
					continue
				}

				node.Metadata[prop.Name] = strVal
			} else if prop.DataType == DataTypeBytes {
				strVal, ok := val.(string)

				if !ok {
					klog.V(1).Infof(
						"Unable to extract memory prop [%s] from [%s.%s] Name: [%s] since it's not a string: %v",
						prop.Name, kind, group, r.GetName(), val,
					)
					continue
				}

				mem, memErr := memoryToBytes(strVal)
				if memErr != nil {
					klog.V(1).Infof(
						"Unable to parse memory value [%s] from [%s.%s] Name: [%s] Reason: %v",
						strVal, kind, group, r.GetName(), memErr,
					)
				}
				node.Properties[prop.Name] = mem
			} else if prop.DataType == DataTypeSelector {
				if m, ok := val.(map[string]interface{}); ok {
					selector := make(map[string]string, len(m))
					for k, v := range m {
						switch t := v.(type) {
						case string:
							selector[k] = t
						case bool:
							selector[k] = strconv.FormatBool(t)
						case int:
							selector[k] = strconv.Itoa(t)
						default:
							klog.V(1).Infof("Parsed unsupported type [%T] from [%s.%s] building selector map for Name: %s", t, kind, group, r.GetName())
						}
					}
					node.Properties[prop.Name] = selector
				} else {
					klog.V(1).Infof("Unable to parse selector value [%v] from [%s.%s] Name: [%s]", val, kind, group, r.GetName())
				}
			} else {
				node.Properties[prop.Name] = val
			}
		} else {
			// path is valid but has no values, e.g. {status: {conditions: []}} where JSONPath == {.status.conditions[?(@.type=="AgentConnected")].status}
			klog.V(3).Infof("Extracting [%s] from [%s.%s] Name: [%s] returned no values",
				prop.Name, kind, group, r.GetName())
			continue
		}
	}

	if found {
		klog.V(5).Infof("Built [%s.%s] using transform config.\nNode: %+v\n", kind, group, node)
	}

	return node
}

func commonStatusConditions(kind string, group string, r *unstructured.Unstructured) map[string]string {
	conditionsMap := make(map[string]string, 0)
	if group != "" {
		group = "." + group
	}

	// FIXME: when on Go 1.24 get via https://pkg.go.dev/sigs.k8s.io/cluster-api@v1.10.4/api/v1beta1#Conditions to simplify

	jp := jsonpath.New("conditions")
	parseErr := jp.Parse(`{.status.conditions}`)
	if parseErr != nil {
		klog.Errorf("Error parsing jsonpath [{.status.conditions}] for [%s.%s] prop: [conditions]. Reason: %v",
			kind, group, parseErr)
	}

	result, err := jp.FindResults(r.Object)
	if err != nil {
		// This error isn't always indicative of a problem, for example, when the object is created, it
		// won't have a status yet, so the jsonpath returns an error until controller adds the status.
		klog.V(4).Infof("Unable to extract prop [condition] from [%s.%s] Name: [%s]. Reason: %v",
			kind, group, r.GetName(), err)
	}
	if len(result) > 0 && len(result[0]) > 0 {
		val := result[0][0].Interface()
		conditions, ok := val.([]interface{})
		if !ok {
			klog.V(1).Infof("Unable to extract prop [condition] from [%s.%s] Name: [%s] since it's not []interface: %v",
				kind, group, r.GetName(), val)
		}
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := condMap["type"].(string)
			status, _ := condMap["status"].(string)
			conditionsMap[condType] = status
		}
	}
	return conditionsMap
}

func memoryToBytes(memory string) (int64, error) {
	quantity, err := resource.ParseQuantity(memory)
	if err != nil {
		return 0, err
	}
	return quantity.Value(), nil
}
