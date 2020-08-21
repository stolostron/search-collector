// Copyright (c) 2020 Red Hat, Inc.

package transforms

import (
	"strings"
	"time"

	"github.com/open-cluster-management/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type GenericResource struct {
	node Node
}

// Builds a GenericResource node. Extract the useful properties from unstructured resource.
func GenericResourceBuilder(r *unstructured.Unstructured) *GenericResource {
	n := Node{
		UID:        prefixedUID(r.GetUID()),
		Properties: unstructuredProperties(r),
		Metadata:   make(map[string]string),
	}
	n.Metadata["OwnerUID"] = ownerRefUID(r.GetOwnerReferences())

	return &GenericResource{node: n}
}

// Need to keep this for compatibility. Node is now computed on the "constructor" GenericResourceBuilder()
func (r GenericResource) BuildNode() Node {
	return r.node
}

func (u GenericResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}

// TODO: Consolidate with commonProperties() in common.go
// Extracts the common properties from any k8s resource and returns them in a map ready to be put in an Node
func unstructuredProperties(r *unstructured.Unstructured) map[string]interface{} {
	ret := make(map[string]interface{})

	ret["kind"] = r.GetKind()
	ret["name"] = r.GetName()
	ret["selfLink"] = r.GetSelfLink()
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
