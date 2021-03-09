// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

	app "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

// ArgoApplicationResource ...
type ArgoApplicationResource struct {
	node Node
}

// ArgoApplicationResourceBuilder ...
func ArgoApplicationResourceBuilder(a *app.Application) *ArgoApplicationResource {
	node := transformCommon(a)
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type
	if a.Spec.Destination != nil {
		if a.Spec.Destination.Name != "" {
			node.Properties["destinationName"] = a.Spec.Destination.Name
		}
		if a.Spec.Destination.Namespace != "" {
			node.Properties["destinationNamespace"] = a.Spec.Destination.Namespace
		}
		if a.Spec.Destination.Server != "" {
			node.Properties["destinationServer"] = a.Spec.Destination.Server
		}
	}
	if a.Spec.Source != nil {
		if a.Spec.Source.Path != "" {
			node.Properties["path"] = a.Spec.Source.Path
		}
		if a.Spec.Source.Chart != "" {
			node.Properties["chart"] = a.Spec.Source.Chart
		}
		if a.Spec.Source.RepoURL != "" {
			node.Properties["repoURL"] = a.Spec.Source.RepoURL
		}
		if a.Spec.Source.TargetRevision != "" {
			node.Properties["targetRevision"] = a.Spec.Source.TargetRevision
		}
	}

	return &ArgoApplicationResource{node: node}
}

// BuildNode construct the node for the Application Resources
func (a ArgoApplicationResource) BuildNode() Node {
	return a.node
}

// BuildEdges construct the edges for the Application Resources
// See documentation at pkg/transforms/README.md
func (a ArgoApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	return ret
}
