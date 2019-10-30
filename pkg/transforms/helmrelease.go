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
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type HelmReleaseResource struct {
	*v1.ConfigMap    // ConfigMap Resource created by Tiller that contains Release info
	*release.Release // Release from Tiller
}

func GetHelmReleaseUID(releaseName string) string {
	return config.Cfg.ClusterName + "/Release/" + releaseName
}

func (h HelmReleaseResource) BuildNode() Node {
	releaseLabels := h.GetLabels()
	releaseName := releaseLabels["NAME"]

	node := Node{
		UID:        GetHelmReleaseUID(releaseName),
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]string),
	}
	// Extract the properties specific to this type
	node.Properties["kind"] = "Release"
	node.Properties["name"] = releaseName
	node.Properties["status"] = releaseLabels["STATUS"]
	revision, err := strconv.ParseInt(releaseLabels["VERSION"], 0, 64)
	if err != nil {
		node.Properties["revision"] = releaseLabels["VERSION"]
	} else {
		node.Properties["revision"] = revision
	}

	if config.Cfg.DeployedInHub {
		node.Properties["_hubClusterResource"] = true
	} else {
		node.Properties["_clusterNamespace"] = config.Cfg.ClusterNamespace
	}

	if h.Release != nil {
		lastDeployed := h.Release.GetInfo().GetLastDeployed()
		timestamp, _ := ptypes.Timestamp(lastDeployed)
		node.Properties["chartName"] = h.Release.GetChart().GetMetadata().GetName()
		node.Properties["chartVersion"] = h.Release.GetChart().GetMetadata().GetVersion()
		node.Properties["namespace"] = h.Release.GetNamespace()
		node.Properties["updated"] = timestamp.UTC().Format(time.RFC3339)
	}
	return node
}

type SummarizedManifestResource struct {
	Name string
	Kind string
}

func getSummarizedManifestResources(h HelmReleaseResource) []SummarizedManifestResource {

	smr := []SummarizedManifestResource{}

	if h.Release == nil {
		glog.V(2).Infof("Cannot retrieve manifest from nil Helm Release %s", h.GetLabels()["NAME"])
		return smr // Can't have any resources without the Release
	}

	/*
		A manifest is a YAML-encoded representation of the Kubernetes resources
		that were generated from this release's chart(s), separated by `---`
		(1) https://helm.sh/docs/helm/#helm-get-manifest
		(2) https://helm.sh/docs/chart_template_guide/#a-first-template
	*/

	manifest := h.Release.GetManifest()

	// Strings for parsing out important information from manifest resources
	kindstr := "kind: "
	namestr := "name: "
	newline := "\n"

	manifestParts := strings.Split(manifest, "---\n") // Split manifest yaml into multiple resource yamls.

	for _, resource := range manifestParts { //	Per resource yaml ...

		kind := ""
		if strings.Contains(resource, kindstr) { // ... if resource kind defined...
			kindstart := strings.Index(resource, kindstr) + len(kindstr)
			kindstop := strings.Index(resource[kindstart:], newline) + kindstart
			kind = resource[kindstart:kindstop] // ... pull `KIND` value out of `kind: KIND` line ...
			if strings.Contains(kind, "'") || strings.Contains(kind, "\r") {
				kind = strings.TrimRight(strings.TrimLeft(kind, "'\r"), "'\r") // ... remove surrounding single quotes '' or carriage returns \r, if any
			}
			kind = strings.TrimSpace(kind)
		}

		name := ""
		if strings.Contains(resource, namestr) { // ... and if resource name defined...
			namestart := strings.Index(resource, namestr) + len(namestr)
			namestop := strings.Index(resource[namestart:], newline) + namestart
			name = resource[namestart:namestop]        // ... pull `"NAME"` value out of `name: "NAME"` line...
			name = strings.Replace(name, "\"", "", -1) // ... and remove surrounding "" from `"NAME"`
			if strings.Contains(name, "'") || strings.Contains(name, "\r") {
				name = strings.TrimRight(strings.TrimLeft(name, "'\r"), "'\r") // ... remove surrounding single quotes '' or carriage returns \r, if any
			}
			name = strings.TrimSpace(name)
		}

		if name != "" && kind != "" { // ... and if both resource kind and name defined...
			smr = append(smr, SummarizedManifestResource{name, kind}) // ... prep `KIND` and `NAME` for BuildEdges
		}
	}

	return smr
}

func (h HelmReleaseResource) BuildEdges(ns NodeStore) []Edge {

	smr := getSummarizedManifestResources(h)

	UID := GetHelmReleaseUID(h.GetLabels()["NAME"])
	edges := []Edge{}
	helmNode := ns.ByUID[UID]

	for _, resource := range smr {
		namespace := h.GetNamespace()
		kind := resource.Kind

		//Obtain Read Lock before checking the map
		NonNSResMapMutex.RLock()
		_, notNameSpaced := NonNSResourceMap[kind]
		NonNSResMapMutex.RUnlock()

		if notNameSpaced {
			// These are non-namespaced resources. So check in namespace "_NONE"
			namespace = "_NONE"
		}

		// ownedBy edges
		if resourceNode, ok := ns.ByKindNamespaceName[kind][namespace][resource.Name]; ok {
			if resourceNode.Metadata != nil { // Metadata can be nil if no node found
				resourceNode.Metadata["ReleaseUID"] = GetHelmReleaseUID(h.GetLabels()["NAME"]) // update node metadata to include release for upstream edge from resource to Release
			}
			if GetHelmReleaseUID(h.GetLabels()["NAME"]) != "" {
				// Add hosting Subscription/Deployable properties to the resource so that they can tracked
				if helmNode.Properties["_hostingSubscription"] != "" || helmNode.Properties["_hostingDeployable"] != "" {
					resourceNode := ns.ByUID[resourceNode.UID]
					//Copy the properties only if the node doesn't have it yet or if they are not the same
					if _, ok := resourceNode.Properties["_hostingSubscription"]; !ok && helmNode.Properties["_hostingSubscription"] != resourceNode.Properties["_hostingSubscription"] {
						copyhostingSubProperties(UID, resourceNode.UID, ns)
					}
				}
				if resourceNode.UID != GetHelmReleaseUID(h.GetLabels()["NAME"]) { //avoid connecting node to itself
					edges = append(edges, Edge{
						SourceUID: resourceNode.UID,
						DestUID:   GetHelmReleaseUID(h.GetLabels()["NAME"]),
						EdgeType:  "ownedBy",
					})
				}
			} else {
				glog.V(2).Infof("%s/%s edge ownedBy Helm Release not created: Helm Release %s not found", kind, resource.Name, h.GetLabels()["NAME"])
			}
		} else {
			glog.V(2).Infof("edge ownedBy Helm Release %s not created: Resource %s/%s not found in namespace %s", h.GetLabels()["NAME"], kind, resource.Name, namespace)
		}
	}

	nodeInfo := NodeInfo{NameSpace: h.GetNamespace(), UID: UID, Kind: h.Kind, Name: h.GetLabels()["NAME"]}
	//deployer subscriber edges
	edges = append(edges, edgesByDeployerSubscriber(nodeInfo, ns)...)

	return edges
}
