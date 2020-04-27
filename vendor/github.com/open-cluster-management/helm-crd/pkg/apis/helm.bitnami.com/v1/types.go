package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:noStatus

// HelmRelease describes a Helm chart release.
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

// HelmReleaseSpec is the spec for a HelmRelease resource.
type HelmReleaseSpec struct {
	// RepoURL is the URL of the repository. Defaults to stable repo.
	RepoURL string `json:"repoUrl,omitempty"`
	// HelmRepoConfigRef contains client configuration to connect to the helm repo
	HelmRepoConfigRef *corev1.ObjectReference `json:"helmRepoConfigRef,omitempty"`
	// ChartName is the name of the chart within the repo
	ChartName string `json:"chartName,omitempty"`
	// ReleaseName is the Name of the release given to Tiller. Defaults to namespace-name. Must not be changed after initial object creation.
	ReleaseName string `json:"releaseName,omitempty"`
	// Version is the chart version
	Version string `json:"version,omitempty"`
	// Auth is the authentication
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`
	// Values is a string containing (unparsed) YAML values
	Values string `json:"values,omitempty"`
}

//HelmReleaseStatus of a helmRelease.
type HelmReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=Success,Failure
	Status         string      `json:"status,omitempty"`
	Message        string      `json:"message,omitempty"`
	Reason         string      `json:"reason,omitempty"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmReleaseList is a list of HelmRelease resources
type HelmReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HelmRelease `json:"items"`
}
