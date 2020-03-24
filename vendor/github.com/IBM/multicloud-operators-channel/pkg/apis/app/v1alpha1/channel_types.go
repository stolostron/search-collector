// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var (
	// KeyChannelSource is namespacedname tells the source of the deployable
	KeyChannelSource = SchemeGroupVersion.Group + "/hosting-deployable"
	// KeyChannel is namespacedname tells the source of the channel
	KeyChannel = SchemeGroupVersion.Group + "/channel"
	// ServingChannel tells the channel that the secrect or configMap refers to
	ServingChannel = SchemeGroupVersion.Group + "/serving-channel"
)

// ChannelType defines types of channel
type ChannelType string

const (
	// ChannelTypeNamespace defines type name of namespace channel
	ChannelTypeNamespace = "namespace"
	// ChannelTypeHelmRepo defines type name of helm repository channel
	ChannelTypeHelmRepo = "helmrepo"
	// ChannelTypeObjectBucket defines type name of bucket in object store
	ChannelTypeObjectBucket = "objectbucket"
	// ChannelTypeGitHub defines type name of GitHub repository
	ChannelTypeGitHub = "github"
)

// ChannelGate defines criteria for promote to channel
type ChannelGate struct {
	Name          string                `json:"name,omitempty"`
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Annotations   map[string]string     `json:"annotations,omitempty"`
}

// ChannelSpec defines the desired state of Channel
type ChannelSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum={Namespace,HelmRepo,ObjectBucket,GitHub,namespace,helmrepo,objectbucket,github}
	Type         ChannelType             `json:"type"`
	PathName     string                  `json:"pathname"`
	SecretRef    *corev1.ObjectReference `json:"secretRef,omitempty"`
	ConfigMapRef *corev1.ObjectReference `json:"configRef,omitempty"`
	Gates        *ChannelGate            `json:"gates,omitempty"`
	// +listType=set
	SourceNamespaces []string `json:"sourceNamespaces,omitempty"`
}

// ChannelStatus defines the observed state of Channel
type ChannelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Channel is the Schema for the channels API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="type of the channel"
// +kubebuilder:printcolumn:name="PathName",type="string",JSONPath=".spec.pathname",description="pathname of the channel"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced
type Channel struct {
	Status            ChannelStatus `json:"status,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ChannelSpec `json:"spec,omitempty"`
	metav1.TypeMeta   `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChannelList contains a list of Channel
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []Channel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Channel{}, &ChannelList{})
}
