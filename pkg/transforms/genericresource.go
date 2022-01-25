// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenericResource ...
type GenericResource struct {
	node Node
}

// GenericResourceBuilder ...
// Builds a GenericResource node. Extract the useful properties from unstructured resource.
func GenericResourceBuilder(r *unstructured.Unstructured) *GenericResource {
	n := Node{
		UID:        prefixedUID(r.GetUID()),
		Properties: unstructuredProperties(r),
		Metadata:   make(map[string]string),
	}
	n.Metadata["OwnerUID"] = ownerRefUID(r.GetOwnerReferences())
	//Adding OwnerReleaseName and Namespace for the resources that doesn't have ownerRef, but are deployed by a release
	if n.Metadata["OwnerUID"] == "" && r.GetAnnotations()["meta.helm.sh/release-name"] != "" &&
		r.GetAnnotations()["meta.helm.sh/release-namespace"] != "" {
		n.Metadata["OwnerReleaseName"] = r.GetAnnotations()["meta.helm.sh/release-name"]
		n.Metadata["OwnerReleaseNamespace"] = r.GetAnnotations()["meta.helm.sh/release-namespace"]
	}
	return &GenericResource{node: n}
}

// BuildNode construct the node for Generic Resources
// Need to keep this for compatibility. Node is now computed on the "constructor" GenericResourceBuilder()
func (r GenericResource) BuildNode() Node {
	return r.node
}

// BuildEdges construct the edges for Generic Resources
func (r GenericResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}

// TODO: Consolidate with commonProperties() in common.go
// Extracts the common properties from any k8s resource and returns them in a map ready to be put in an Node
func unstructuredProperties(r *unstructured.Unstructured) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["kind"] = r.GetKind()
	ret["name"] = r.GetName()
	ret["created"] = r.GetCreationTimestamp().UTC().Format(time.RFC3339)
	ret["_clusterNamespace"] = config.Cfg.ClusterNamespace
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
