/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HelmReleaseStatusEnum defines the status of a Subscription release
type HelmReleaseStatusEnum string

const (
	// HelmReleaseFailed means this subscription is the "parent" sitting in hub
	HelmReleaseFailed HelmReleaseStatusEnum = "Failed"
	// HelmReleaseSuccess means this subscription is the "parent" sitting in hub
	HelmReleaseSuccess HelmReleaseStatusEnum = "Success"
)

//SourceTypeEnum types of sources
type SourceTypeEnum string

const (
	// HelmRepoSourceType helmrepo source type
	HelmRepoSourceType SourceTypeEnum = "helmrepo"
	// GitHubSourceType github source type
	GitHubSourceType SourceTypeEnum = "github"
)

//HelmReleaseStatus struct containing the status
type HelmReleaseStatus struct {
	Status         HelmReleaseStatusEnum `json:"phase,omitempty"`
	Message        string                `json:"message,omitempty"`
	Reason         string                `json:"reason,omitempty"`
	LastUpdateTime metav1.Time           `json:"lastUpdate"`
}

//GitHub provides the parameters to access the helm-chart located in a github repo
type GitHub struct {
	Urls      []string `json:"urls,omitempty"`
	ChartPath string   `json:"chartPath,omitempty"`
	Branch    string   `json:"branch,omitempty"`
}

//HelmRepo provides the urls to retrieve the helm-chart
type HelmRepo struct {
	Urls []string `json:"urls,omitempty"`
}

//Source holds the different types of repository
type Source struct {
	SourceType SourceTypeEnum `json:"type,omitempty"`
	GitHub     *GitHub        `json:"github,omitempty"`
	HelmRepo   *HelmRepo      `json:"helmRepo,omitempty"`
}

func (s Source) String() string {
	switch strings.ToLower(string(s.SourceType)) {
	case string(HelmRepoSourceType):
		return fmt.Sprintf("%v", s.HelmRepo.Urls)
	case string(GitHubSourceType):
		return fmt.Sprintf("%v|%s|%s", s.GitHub.Urls, s.GitHub.Branch, s.GitHub.ChartPath)
	default:
		return fmt.Sprintf("SourceType %s not supported", s.SourceType)
	}
}

// HelmReleaseSpec defines the desired state of HelmRelease
// +k8s:openapi-gen=true
type HelmReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// Source holds the url toward the helm-chart
	Source *Source `json:"source,omitempty"`
	// ChartName is the name of the chart within the repo
	ChartName string `json:"chartName,omitempty"`
	// Version is the chart version
	Version string `json:"version,omitempty"`
	// Values is a string containing (unparsed) YAML values
	Values string `json:"values,omitempty"`
	// Secret to use to access the helm-repo defined in the CatalogSource.
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`
	// Configuration parameters to access the helm-repo defined in the CatalogSource
	ConfigMapRef *corev1.ObjectReference `json:"configMapRef,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmRelease is the Schema for the subscriptionreleases API
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec,omitempty"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmReleaseList contains a list of HelmRelease
type HelmReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmRelease{}, &HelmReleaseList{})
}
