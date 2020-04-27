// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package v1alpha1

import (
	policy "github.com/open-cluster-management/hcm-compliance/pkg/apis/policy/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition is the base struct for representing resource conditions
type Condition struct {
	// Type of condition, e.g Complete or Failed.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// Kind of Compliance
const ComplianceKind = "Compliance"

// ComplianceResourcePlural as a plural name
const ComplianceResourcePlural = "compliances"

// Compliance is a specification for a Compliance resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type Compliance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ComplianceSpec `json:"spec"`

	Status ComplianceStatus `json:"status,omitempty"`

	//PerClusterStatus CompliancePerClusterStatus `json:"perClusterStatus,omitempty"`
}

// ComplianceSpec is the spec for a Compliance resource
type ComplianceSpec struct {

	// +optional
	Ignore bool `json:"ignore,omitempty"`

	RuntimeRules []policy.Policy `json:"runtime-rules,omitempty"`

	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []Condition `json:"conditions,omitempty"`
}

//CompliancePerClusterStatus indicating whether it is compliant in its own cluster not globally
type CompliancePerClusterStatus struct {
	AggregatePolicyStatus map[string]*policy.PolicyStatus `json:"aggregatePoliciesStatus,omitempty"`
	ComplianceState       policy.ComplianceState          `json:"compliant,omitempty"`
	ClusterName           string                          `json:"clustername,omitempty"`
}

type ComplianceMap map[string]*CompliancePerClusterStatus

//ComplianceStatus indicating whether it is compliant or not globally
type ComplianceStatus struct {
	Status            ComplianceMap `json:"status,omitempty"`
	PlacementPolicies []string      `json:"placementPolicies,omitempty"`
	PlacementBindings []string      `json:"placementBindings,omitempty"`
}

// ComplianceList is a list of Compliance resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:lister-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ComplianceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Compliance `json:"items"`
}

// ComplianceScope where the compliance applies
type ComplianceScope struct {
	Clusters  string   `json:"clusters,omitempty" protobuf:"bytes,5,rep,name=clusters"`
	Namespace []string `json:"namespace,omitempty" protobuf:"bytes,5,rep,name=namespace"`
}
