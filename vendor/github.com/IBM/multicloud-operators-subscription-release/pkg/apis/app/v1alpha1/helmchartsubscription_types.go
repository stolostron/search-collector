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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

//ChartsDir env variable name which contains the directory where the charts are installed
const ChartsDir = "CHARTS_DIR"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PackageFilter defines the reference to Channel
type PackageFilter struct {
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Annotations   map[string]string     `json:"annotations,omitempty"`
	// +kubebuilder:validation:Pattern=([0-9]+)((\.[0-9]+)(\.[0-9]+)|(\.[0-9]+)?(\.[xX]))$
	Version string `json:"version,omitempty"`
}

// PackageOverride describes rules for override
type PackageOverride struct {
	runtime.RawExtension `json:",inline"`
}

// Overrides field in deployable
type Overrides struct {
	PackageName string `json:"packageName"`
	// +kubebuilder:validation:MinItems=1
	PackageOverrides []PackageOverride `json:"packageOverrides"` // To be added
}

//GitHubSubscription provides information to retrieve the helm-chart from github
type GitHubSubscription struct {
	Urls       []string `json:"urls,omitempty"`
	ChartsPath string   `json:"chartsPath,omitempty"`
	Branch     string   `json:"branch,omitempty"`
}

//HelmRepoSubscription provides the urls to retrieve the helm-chart
type HelmRepoSubscription struct {
	Urls []string `json:"urls,omitempty"`
}

//SourceSubscription holds the different types of repository
type SourceSubscription struct {
	SourceType SourceTypeEnum        `json:"type,omitempty"`
	GitHub     *GitHubSubscription   `json:"github,omitempty"`
	HelmRepo   *HelmRepoSubscription `json:"helmRepo,omitempty"`
}

func (s SourceSubscription) String() string {
	switch strings.ToLower(string(s.SourceType)) {
	case string(HelmRepoSourceType):
		return fmt.Sprintf("%v", s.HelmRepo.Urls)
	case string(GitHubSourceType):
		return fmt.Sprintf("%v|%s|%s", s.GitHub.Urls, s.GitHub.Branch, s.GitHub.ChartsPath)
	default:
		return fmt.Sprintf("SourceType %s not supported", s.SourceType)
	}
}

// HelmChartSubscriptionSpec defines the desired state of HelmChartSubscription
//// +k8s:openapi-gen=true
type HelmChartSubscriptionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// RepoURL is the URL of the repository. Defaults to stable repo.
	// Source holds the url toward the helm-chart
	Source *SourceSubscription `json:"chartsSource,omitempty"`

	Channel string `json:"channel"`
	// To specify 1 package in channel
	Package string `json:"name,omitempty"`

	InstallPlanApproval Approval `json:"installPlanApproval,omitempty"`

	// To specify more than 1 package in channel
	PackageFilter *PackageFilter `json:"packageFilter,omitempty"`
	// To provide flexibility to override package in channel with local input
	PackageOverrides []*Overrides `json:"packageOverrides,omitempty"`
	// For hub use only, to specify which clusters to go to
	//	Placement *placementv1alpha1.Placement `json:"placement,omitempty"`
	// Secret to use to access the helm-repo defined in the CatalogSource.
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`
	// Configuration parameters to access the helm-repo defined in the CatalogSource
	ConfigMapRef *corev1.ObjectReference `json:"configRef,omitempty"`
}

//Approval approval types
type Approval string

const (
	//ApprovalManual when set to this value, the helmRelease will not be automatically updated if a new version of the helm-chart is available.
	ApprovalManual Approval = "Manual"
	//ApprovalAutomatic when set to this value, the helmRelease will be automatically updated when a new version of the helm-chart is available.
	ApprovalAutomatic Approval = "Automatic"
)

// HelmChartSubscriptionStatusEnum defines the status of a HelmChartSubscription
type HelmChartSubscriptionStatusEnum string

const (
	// HelmChartSubscriptionSuccess means this subscription Succeed
	HelmChartSubscriptionSuccess HelmChartSubscriptionStatusEnum = "Success"
	// HelmChartSubscriptionFailed means this subscription Failed
	HelmChartSubscriptionFailed HelmChartSubscriptionStatusEnum = "Failed"
)

// HelmChartSubscriptionUnitStatus defines status of a unit (subscription or package)
type HelmChartSubscriptionUnitStatus struct {
	// Phase are Propagated if it is in hub or Subscribed if it is in endpoint
	Status         HelmChartSubscriptionStatusEnum `json:"status,omitempty"`
	Message        string                          `json:"message,omitempty"`
	Reason         string                          `json:"reason,omitempty"`
	LastUpdateTime metav1.Time                     `json:"lastUpdateTime"`
}

// HelmChartSubscriptionStatus defines the observed state of HelmChartSubscription
//// +k8s:openapi-gen=true
type HelmChartSubscriptionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	HelmChartSubscriptionUnitStatus `json:",inline"`

	HelmChartSubscriptionPackageStatus map[string]HelmChartSubscriptionUnitStatus `json:"packages,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmChartSubscription is the Schema for the subscriptions API
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
type HelmChartSubscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmChartSubscriptionSpec   `json:"spec,omitempty"`
	Status HelmChartSubscriptionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmChartSubscriptionList contains a list of HelmChartSubscription
type HelmChartSubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmChartSubscription `json:"items"`
}

// Subscriber defines the interface for various channels
type Subscriber interface {
	Restart() error
	Stop() error
	Update(*HelmChartSubscription) error
	IsStarted() bool
}

func init() {
	SchemeBuilder.Register(&HelmChartSubscription{}, &HelmChartSubscriptionList{})
}
