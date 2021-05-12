// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoApplicationResource ...
type ArgoApplicationResource struct {
	node Node
}

type ArgoApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ArgoApplicationSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            ArgoApplicationStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type ArgoApplicationSpec struct {
	Source      ArgoApplicationSource      `json:"source" protobuf:"bytes,1,opt,name=source"`
	Destination ArgoApplicationDestination `json:"destination" protobuf:"bytes,2,name=destination"`
}

type ArgoApplicationSource struct {
	RepoURL        string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	Path           string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
	TargetRevision string `json:"targetRevision,omitempty" protobuf:"bytes,4,opt,name=targetRevision"`
	Chart          string `json:"chart,omitempty" protobuf:"bytes,12,opt,name=chart"`
}

type ArgoApplicationDestination struct {
	Server    string `json:"server,omitempty" protobuf:"bytes,1,opt,name=server"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	Name      string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
}

type ArgoApplicationStatus struct {
	Health ArgoApplicationHealth `json:"health" protobuf:"bytes,1,opt,name=health"`
}

type ArgoApplicationHealth struct {
	Status string `json:"status" protobuf:"bytes,1,opt,name=status"`
}

// ArgoApplicationResourceBuilder ...
func ArgoApplicationResourceBuilder(a *ArgoApplication) *ArgoApplicationResource {
	node := transformCommon(a)
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type

	// ApplicationSet
	applicationSet := ""
	for _, ref := range a.ObjectMeta.OwnerReferences {
		if strings.Index(ref.APIVersion, "argoproj.io/") == 0 && ref.Kind == "ApplicationSet" {
			applicationSet = ref.Name
			break
		}
	}
	node.Properties["applicationSet"] = applicationSet

	// Destination properties
	node.Properties["destinationName"] = a.Spec.Destination.Name
	node.Properties["destinationNamespace"] = a.Spec.Destination.Namespace
	node.Properties["destinationServer"] = a.Spec.Destination.Server

	// Source properties
	node.Properties["path"] = a.Spec.Source.Path
	node.Properties["chart"] = a.Spec.Source.Chart
	node.Properties["repoURL"] = a.Spec.Source.RepoURL
	node.Properties["targetRevision"] = a.Spec.Source.TargetRevision

	// Status properties
	node.Properties["status"] = a.Status.Health.Status

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
