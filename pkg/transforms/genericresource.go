// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"fmt"
	"strings"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
)

// GenericResource ...
type GenericResource struct {
	node Node
}

// Builds a GenericResource node.
// Extract default properties from unstructured resource.
// Supports extracting additional properties defined by the transform config.
func GenericResourceBuilder(r *unstructured.Unstructured) *GenericResource {
	n := Node{
		UID:        prefixedUID(r.GetUID()),
		Properties: genericProperties(r),
		Metadata:   genericMetadata(r),
	}

	// Check if a transform config exists for this resource and extract the additional properties.
	transformConfig, found := getTransformConfig(r.GroupVersionKind().Group, r.GetKind())
	if found {
		for _, prop := range transformConfig.properties {
			jp := jsonpath.New(prop.name)
			parseErr := jp.Parse(prop.jsonpath)
			if parseErr != nil {
				klog.Errorf("Error parsing jsonpath %s for property %s: %v", prop.jsonpath, prop.name, parseErr)
				continue
			}
			result, err := jp.FindResults(r.Object)
			if err != nil {
				klog.Errorf("Error extracting property %s from resource %s: %v", prop.name, r.GetName(), err)
				continue
			}
			// klog.Infof("%s\t=> %s\t%s", r.GroupVersionKind(), prop.name, result[0][0])
			n.Properties[prop.name] = fmt.Sprintf("%s", result[0][0])
		}
		klog.V(5).Infof("Generic resource [%s.%s] built using transform config.\n%+v\n\n", r.GetKind(), r.GroupVersionKind().Group, n)
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
func genericProperties(r *unstructured.Unstructured) map[string]interface{} {
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

func genericMetadata(r *unstructured.Unstructured) map[string]string {
	metadata := make(map[string]string)
	metadata["OwnerUID"] = ownerRefUID(r.GetOwnerReferences())
	//Adding OwnerReleaseName and Namespace for the resources that doesn't have ownerRef, but are deployed by a release
	if metadata["OwnerUID"] == "" && r.GetAnnotations()["meta.helm.sh/release-name"] != "" &&
		r.GetAnnotations()["meta.helm.sh/release-namespace"] != "" {
		metadata["OwnerReleaseName"] = r.GetAnnotations()["meta.helm.sh/release-name"]
		metadata["OwnerReleaseNamespace"] = r.GetAnnotations()["meta.helm.sh/release-namespace"]
	}

	return metadata
}
