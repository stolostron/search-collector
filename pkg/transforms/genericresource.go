// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
)

// GenericResource ...
type GenericResource struct {
	node         Node
	extractEdges []ExtractEdge
	r            *unstructured.Unstructured // TODO: remove r from here
}

// Builds a GenericResource node.
// Extract default properties from unstructured resource.
// Supports extracting additional properties defined by the transform config.
func GenericResourceBuilder(r *unstructured.Unstructured, additionalColumns ...ExtractProperty) *GenericResource {
	n := Node{
		UID:        prefixedUID(r.GetUID()),
		Properties: genericProperties(r),
		Metadata:   genericMetadata(r),
	}

	n = applyDefaultTransformConfig(n, r, additionalColumns...)

	tc, ok := getTransformConfig(r.GroupVersionKind().Group, r.GetKind())
	if !ok || len(tc.edges) == 0 {
		return &GenericResource{node: n}
	}

	return &GenericResource{node: n, extractEdges: tc.edges, r: r} // TODO: remove r from here
}

// BuildNode construct the node for Generic Resources
// Need to keep this for compatibility. Node is now computed on the "constructor" GenericResourceBuilder()
func (r GenericResource) BuildNode() Node {
	return r.node
}

// BuildEdges construct the edges for Generic Resources
func (r GenericResource) BuildEdges(ns NodeStore) []Edge {
	if len(r.extractEdges) > 0 {
		klog.Infof("Building edges from config for kind: %s", r.node.Properties["kind"])
		edges := []Edge{}
		for _, edge := range r.extractEdges {

			// Resolving the edge
			jp := jsonpath.New("name")
			parseErr := jp.Parse(edge.Name)
			if parseErr != nil {
				klog.Errorf("Error parsing edge.Name from jsonpath: %v", parseErr)
				continue
			}

			result, err := jp.FindResults(r.r.Object)
			if err != nil {
				// This error isn't always indicative of a problem, for example, when the object is created, it
				// won't have a status yet, so the jsonpath returns an error until controller adds the status.
				klog.Errorf("Unable to extract edge.Name from jsonpath: %v", err)
				continue
			}

			if len(result) > 0 && len(result[0]) > 0 {
				val := result[0][0].Interface()

				name := val.(string)
				namespace := r.node.Properties["namespace"].(string)
				sourceKind := r.node.Properties["kind"].(string)
				destUID := ns.ByKindNamespaceName[edge.Kind][namespace][name].UID

				if destUID == "" {
					klog.Infof("No destination found for kind: %s, namespace:%s, name: %s", edge.Kind, namespace, name)
					continue
				}
				edges = append(edges, Edge{
					SourceUID:  r.node.UID,
					SourceKind: sourceKind,
					DestUID:    destUID,
					DestKind:   edge.Kind,
					EdgeType:   edge.EdgeType,
				})
			}
		}
		klog.Infof("Built edges from config. %+v", edges)
		return edges
	}

	return []Edge{}
}

// TODO: Consolidate with commonProperties() in common.go
// Extracts the common properties from any k8s resource and returns them in a map ready to be put in an Node
func genericProperties(r *unstructured.Unstructured) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["kind"] = r.GetKind()
	ret["name"] = r.GetName()
	ret["created"] = r.GetCreationTimestamp().UTC().Format(time.RFC3339)
	if config.Cfg.DeployedInHub {
		ret["_hubClusterResource"] = true
	}

	// valid api group with have format of "apigroup/version"
	// unnamed api groups will have format of "/version"
	slice := strings.Split(r.GetAPIVersion(), "/")
	if len(slice) == 2 {
		ret["apigroup"] = slice[0]
		ret["apiversion"] = slice[1]
	} else {
		ret["apiversion"] = slice[0]
	}

	if r.GetLabels() != nil {
		ret["label"] = r.GetLabels()
	}

	annotations := commonAnnotations(r)

	if annotations != nil {
		ret["annotation"] = annotations
	}

	if r.GetNamespace() != "" {
		ret["namespace"] = r.GetNamespace()
	}
	if r.GetAnnotations()["apps.open-cluster-management.io/hosting-subscription"] != "" {
		ret["_hostingSubscription"] = r.GetAnnotations()["apps.open-cluster-management.io/hosting-subscription"]
	}
	if r.GetAnnotations()["apps.open-cluster-management.io/hosting-deployable"] != "" {
		ret["_hostingDeployable"] = r.GetAnnotations()["apps.open-cluster-management.io/hosting-deployable"]
	}

	return ret
}

func genericMetadata(r *unstructured.Unstructured) map[string]any {
	metadata := make(map[string]any)
	// When a resource is mutated by Gatekeeper, add this annotation
	mutation, ok := r.GetAnnotations()["gatekeeper.sh/mutations"]
	if ok {
		metadata["gatekeeper.sh/mutations"] = mutation
	}

	metadata["OwnerUID"] = ownerRefUID(r.GetOwnerReferences())
	// Adds OwnerReleaseName and Namespace to resources that don't have ownerRef, but are deployed by a release.
	if metadata["OwnerUID"] == "" && r.GetAnnotations()["meta.helm.sh/release-name"] != "" &&
		r.GetAnnotations()["meta.helm.sh/release-namespace"] != "" {
		metadata["OwnerReleaseName"] = r.GetAnnotations()["meta.helm.sh/release-name"]
		metadata["OwnerReleaseNamespace"] = r.GetAnnotations()["meta.helm.sh/release-namespace"]
	}

	return metadata
}
