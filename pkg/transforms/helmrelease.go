/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
*/

package transforms

import (
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/config"
	"gopkg.in/yaml.v3"
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
		Metadata:   make(map[string]any),
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
	}

	if h.Release != nil {
		lastDeployed := h.Release.GetInfo().GetLastDeployed()
		timestamp := lastDeployed.AsTime()
		node.Properties["chartName"] = h.Release.GetChart().GetMetadata().GetName()
		node.Properties["chartVersion"] = h.Release.GetChart().GetMetadata().GetVersion()
		node.Properties["namespace"] = h.Release.GetNamespace()
		node.Properties["updated"] = timestamp.UTC().Format(time.RFC3339)
	}
	return node
}

type SummarizedManifestResource struct {
	Kind     string
	Metadata struct {
		Name string
	}
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

	manifestParts := strings.Split(manifest, "---\n") // Split manifest yaml into multiple resource yamls.

	for _, resource := range manifestParts { //	Per resource yaml ...

		tmpsmr := SummarizedManifestResource{}
		// We unmarshal the struct
		err := yaml.Unmarshal([]byte(resource), &tmpsmr)
		if err != nil {
			glog.Errorf("Unmarshalling Helm Release %s failed: %v", h.GetLabels()["NAME"], err)
		} else if tmpsmr.Kind != "" && tmpsmr.Metadata.Name != "" { // ... and if both resource kind and name defined...
			smr = append(smr, tmpsmr) // ... prep `KIND` and `NAME` for BuildEdges
		} else { // this shouldn't happen
			glog.Warningf("kind or name not found for resource in Helm Release %s", h.GetLabels()["NAME"])
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
		name := resource.Metadata.Name

		// Obtain Read Lock before checking the map
		NonNSResMapMutex.RLock()
		_, notNameSpaced := NonNSResourceMap[kind]
		NonNSResMapMutex.RUnlock()

		if notNameSpaced {
			// These are non-namespaced resources. So check in namespace "_NONE"
			namespace = "_NONE"
		}

		// ownedBy edges
		if resourceNode, ok := ns.ByKindNamespaceName[kind][namespace][name]; ok {
			if resourceNode.Metadata != nil { // Metadata can be nil if no node found
				// update node metadata to include release for upstream edge from resource to Release
				resourceNode.Metadata["ReleaseUID"] = GetHelmReleaseUID(h.GetLabels()["NAME"])
			}
			if GetHelmReleaseUID(h.GetLabels()["NAME"]) != "" {
				// Add hosting Subscription/Deployable properties to the resource so that they can tracked
				if helmNode.Properties["_hostingSubscription"] != "" || helmNode.Properties["_hostingDeployable"] != "" {
					resourceNode := ns.ByUID[resourceNode.UID]
					// Copy the properties only if the node doesn't have it yet or if they are not the same
					if _, ok := resourceNode.Properties["_hostingSubscription"]; !ok &&
						helmNode.Properties["_hostingSubscription"] != resourceNode.Properties["_hostingSubscription"] {
						copyhostingSubProperties(UID, resourceNode.UID, ns)
					}
				}
				if resourceNode.UID != GetHelmReleaseUID(h.GetLabels()["NAME"]) { // avoid connecting node to itself
					edges = append(edges, Edge{
						SourceUID:  resourceNode.UID,
						DestUID:    GetHelmReleaseUID(h.GetLabels()["NAME"]),
						EdgeType:   "ownedBy",
						SourceKind: resourceNode.Properties["kind"].(string),
						DestKind:   "Release",
					})
				}
			} else {
				glog.V(2).Infof("%s/%s edge ownedBy Helm Release not created: Helm Release %s not found",
					kind, name, h.GetLabels()["NAME"])
			}
		} else {
			glog.V(2).Infof("edge ownedBy Helm Release %s not created: Resource %s/%s not found in namespace %s",
				h.GetLabels()["NAME"], kind, name, namespace)
		}
	}
	return edges
}
