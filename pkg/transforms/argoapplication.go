// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"encoding/json"
	"strings"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoApplicationResource ...
type ArgoApplicationResource struct {
	node      Node
	resources []ResourceStatus
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
	Resources      []ResourceStatus       `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	Conditions     []ApplicationCondition `json:"conditions,omitempty" protobuf:"bytes,5,opt,name=conditions"`
	OperationState *OperationState        `json:"operationState,omitempty" protobuf:"bytes,7,opt,name=operationState"`
	Health         HealthStatus           `json:"health" protobuf:"bytes,1,opt,name=health"`
	Sync           SyncStatus             `json:"sync,omitempty" protobuf:"bytes,2,opt,name=sync"`
}

type ResourceStatus struct {
	Group     string        `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Version   string        `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	Kind      string        `json:"kind,omitempty" protobuf:"bytes,3,opt,name=kind"`
	Namespace string        `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	Name      string        `json:"name,omitempty" protobuf:"bytes,5,opt,name=name"`
	Health    *HealthStatus `json:"health,omitempty" protobuf:"bytes,7,opt,name=health"`
}

type ApplicationCondition struct {
	Type    string `json:"type" protobuf:"bytes,1,opt,name=type"`
	Message string `json:"message" protobuf:"bytes,2,opt,name=message"`
}

// OperationState contains information about state of a running operation
type OperationState struct {
	// Phase is the current phase of the operation
	Phase string `json:"phase" protobuf:"bytes,2,opt,name=phase"`
	// Message holds any pertinent messages when attempting to perform operation (typically errors).
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
}

type HealthStatus struct {
	Status string `json:"status" protobuf:"bytes,1,opt,name=status"`
}

type SyncStatus struct {
	Status string `json:"status" protobuf:"bytes,1,opt,name=status,casttype=SyncStatusCode"`
}

// ArgoApplicationResourceBuilder ...
func ArgoApplicationResourceBuilder(a *ArgoApplication) *ArgoApplicationResource {
	node := transformCommon(a)
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type

	// add its ApplicationSet owner
	applicationSet := ""

	for _, ref := range a.ObjectMeta.OwnerReferences {
		if strings.Index(ref.APIVersion, "argoproj.io/") == 0 && ref.Kind == "ApplicationSet" {
			applicationSet = ref.Name

			break
		}
	}

	node.Properties["applicationSet"] = applicationSet

	// add its hosting applicationSet namespaced name, if the argocd app is propogated by ACM argocd pull integration controller
	if a.Annotations != nil {
		if a.Annotations["apps.open-cluster-management.io/hosting-applicationset"] != "" {
			node.Properties["_hostingResource"] = "ApplicationSet/" + a.Annotations["apps.open-cluster-management.io/hosting-applicationset"]
		}
	}

	// Destination properties
	node.Properties["destinationName"] = a.Spec.Destination.Name
	node.Properties["destinationNamespace"] = a.Spec.Destination.Namespace
	node.Properties["destinationServer"] = a.Spec.Destination.Server

	// Source properties
	node.Properties["path"] = a.Spec.Source.Path
	node.Properties["chart"] = a.Spec.Source.Chart
	node.Properties["repoURL"] = a.Spec.Source.RepoURL
	node.Properties["targetRevision"] = a.Spec.Source.TargetRevision

	if a.Spec.Source.TargetRevision == "" {
		node.Properties["targetRevision"] = "HEAD"
	}

	// Status properties
	node.Properties["healthStatus"] = a.Status.Health.Status
	node.Properties["syncStatus"] = a.Status.Sync.Status

	// conditions properties, each condition type is a property
	for _, condition := range a.Status.Conditions {
		truncatedMessage := TruncateText(condition.Message, 512)
		if truncatedMessage > "" {
			node.Properties["_condition"+condition.Type] = truncatedMessage
		}
	}

	// if there is operationState.message, append it to the error condition property
	if a.Status.Health.Status != "Healthy" || a.Status.Sync.Status != "Synced" {
		if a.Status.OperationState != nil && a.Status.OperationState.Message != "" && a.Status.OperationState.Phase != "Succeeded" {
			truncatedMessage := TruncateText(a.Status.OperationState.Message, 512)
			if truncatedMessage > "" {
				node.Properties["_condition"+"OperationError"] = truncatedMessage
			}
		}
	}

	// get resource list and it will be passed to edge
	return &ArgoApplicationResource{node: node, resources: a.Status.Resources}
}

func TruncateText(text string, width int) string {
	if width < 0 {
		glog.Warningf("text truncation width is less than zero, width: %v", width)

		return ""
	}

	res := []rune(text)

	if len(res) > width {
		res = res[:width]

		return string(res) + "..."
	}

	return string(res)
}

// BuildNode construct the node for the Application Resources
func (a ArgoApplicationResource) BuildNode() Node {
	return a.node
}

// BuildEdges construct the edges for the Application Resources
// See documentation at pkg/transforms/README.md
func (a ArgoApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}

	sourceUID := a.node.UID
	sourceKind := a.node.Properties["kind"].(string)

	missingResc := []ResourceStatus{}

	if len(a.resources) > 0 {
		for _, resource := range a.resources {
			namespace := "_NONE"
			if resource.Namespace > "" {
				namespace = resource.Namespace
			}

			if destNode, ok := ns.ByKindNamespaceName[resource.Kind][namespace][resource.Name]; ok {
				if sourceUID != destNode.UID { // avoid connecting node to itself
					ret = append(ret, Edge{
						EdgeType:   "subscribesTo",
						SourceUID:  sourceUID,
						SourceKind: sourceKind,
						DestUID:    destNode.UID,
						DestKind:   resource.Kind,
					})
				}
			} else {
				glog.Warningf("For %s, subscribesTo edge not created as %s named %s not found",
					resource.Kind+"/"+namespace+"/"+resource.Name, resource.Kind, namespace+"/"+resource.Name)

				missingResc = append(missingResc, resource)
			}
		}
	}

	// add missing application resource as a property
	if len(missingResc) != 0 {
		rescb, err := json.Marshal(missingResc)

		if err != nil {
			glog.Error("ArgoApplication transform failed to marshal missingResc", err)
		} else {
			a.node.Properties["_missingResources"] = string(rescb)
		}
	} else {
		delete(a.node.Properties, "_missingResources") // no-op in delete if property doesn't exist
	}

	return ret
}
