// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//ComplianceType describe weather we must or must not have a given resource
type ComplianceType string

const (
	// MustNotHave is an enforcement state to exclude a resource
	MustNotHave ComplianceType = "Mustnothave"

	// MustHave is an enforcement state to include a resource
	MustHave ComplianceType = "Musthave"

	// MustOnlyHave is an enforcement state to exclusively include a resource
	MustOnlyHave ComplianceType = "Mustonlyhave"
)

// RemediationAction describes weather to enforce or inform
type RemediationAction string

const (
	// Enforce is an remediationAction to make changes
	Enforce RemediationAction = "Enforce"

	// Inform is an remediationAction to only inform
	Inform RemediationAction = "Inform"

	// Ignore is an remediationAction to only inform
	Ignore RemediationAction = "Ignore"
)

// EnforcementState shows the state of enforcement //TODO maybe not needed, Type in condition mught be enough
type EnforcementState string

const (
	// SuccessfullyEnforced is an enforcement state to SuccessfullyEnforced
	SuccessfullyEnforced EnforcementState = "SuccessfullyEnforced"

	// FailedToEnforce is an enforcement state to FailedToEnforce
	FailedToEnforce EnforcementState = "FailedtoEnforce"
)

// ComplianceState shows the state of enforcement
type ComplianceState string

const (
	// Compliant is an ComplianceState
	Compliant ComplianceState = "Compliant"

	// NonCompliant is an ComplianceState
	NonCompliant ComplianceState = "NonCompliant"

	// UnknownCompliancy is an ComplianceState
	UnknownCompliancy ComplianceState = "UnknownCompliancy"
)

//ResourceState genric description of a state
type ResourceState string

const (
	// ResourceStateCreated indicates a resource is in a created state
	ResourceStateCreated ResourceState = "Created"
	// ResourceStatePending indicates a resource is in a pending state
	ResourceStatePending ResourceState = "Pending"
	// ResourceStateStopped indicates a resource is in a running state
	ResourceStateStopped ResourceState = "Stopped"
	// ResourceStateFailed indicates a resource is in a failed state
	ResourceStateFailed ResourceState = "Failed"
	// ResourceStateUnknown indicates a resource is in a unknown state
	ResourceStateUnknown ResourceState = "Unknown"
	// ResourceStateDeleting indicates a resource is being deleted
	ResourceStateDeleting ResourceState = "Deleting"
	// ResourceStateOnline indicates a resource has been fully synchronized and online
	ResourceStateOnline ResourceState = "Online"
	// ResourceStateWaiting indicates a resource is in a waiting state, e.g. waiting for dependencies
	ResourceStateWaiting ResourceState = "Waiting"
	// ResourceStateRetrying indicates a resource failed to provision for external reasons. Retrying later on.
	ResourceStateRetrying ResourceState = "Retrying"
)

// Condition is the base struct for representing resource conditions
type Condition struct {
	// Type of condition, e.g Complete or Failed.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
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

// Kind of Policy
const PolicyKind = "Policy"

// PolicyResourcePlural as a plural name
const PolicyResourcePlural = "policies"

// Policy is a specification for a Policy resource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PolicySpec   `json:"spec"`
	Status PolicyStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

// PolicySpec is the spec for a Policy resource
type PolicySpec struct {
	ComplianceType       ComplianceType         `json:"complianceType,omitempty"`    //musthave, mustnothave, mustonlyhave
	RemediationAction    RemediationAction      `json:"remediationAction,omitempty"` //enforce, inform
	Namespaces           Target                 `json:"namespaces,omitempty"`
	RoleTemplates        []*RoleTemplate        `json:"role-templates,omitempty"`
	GenericTemplates     []*GenericTemplate     `json:"generic-templates,omitempty"`
	RoleBindingTemplates []*RoleBindingTemplate `json:"roleBinding-templates,omitempty"`
	ObjectTemplates      []*ObjectTemplate      `json:"object-templates,omitempty"`
	PolicyTemplates      []*PolicyTemplate      `json:"policy-templates,omitempty"`
	Disabled             bool                   `json:"disabled"`

	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []Condition `json:"conditions,omitempty"`

	/*
	  TODO: complete the policy with Rolebinding, NetworkPolicy and so on
	*/
}

//RoleBindingTemplate temple for roleBinding
type RoleBindingTemplate struct {
	// ComplianceType specifies wether it is a : //musthave, mustnothave, mustonlyhave
	ComplianceType ComplianceType `json:"complianceType"`

	//Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`

	// RoleBinding
	RoleBinding rbacv1.RoleBinding `json:"roleBinding"`
	//Status shows the individual status of each template within a policy
	Status TemplateStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

//RoleBindingTemplate temple for roleBinding
type ObjectTemplate struct {
	// ComplianceType specifies wether it is a : //musthave, mustnothave, mustonlyhave
	ComplianceType ComplianceType `json:"complianceType"`

	//Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`

	// RoleBinding
	ObjectDefinition runtime.RawExtension `json:"objectDefinition,omitempty"`
	//Status shows the individual status of each template within a policy
	Status TemplateStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

//PolicyTemplate template for custom security policy
type PolicyTemplate struct {
	ObjectDefinition runtime.RawExtension `json:"objectDefinition,omitempty"`
	//Status shows the individual status of each template within a policy
	Status TemplateStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

// Target defines the list of namespaces to include/exclude
type Target struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type CompliancePerClusterStatus struct {
	AggregatePolicyStatus map[string]*PolicyStatus `json:"aggregatePoliciesStatus,omitempty"`
	ComplianceState       ComplianceState          `json:"compliant,omitempty"`
	ClusterName           string                   `json:"clustername,omitempty"`
}

type ComplianceMap map[string]*CompliancePerClusterStatus

// PolicyStatus is the status for a Policy resource
type PolicyStatus struct {
	ComplianceState ComplianceState `json:"compliant,omitempty"` // Compliant, NonCompliant, UnkownCompliancy

	Valid bool `json:"valid,omitempty"` // a policy can be invalid if it has conflicting roles

	// A human readable message indicating details about why the policy is in this state.
	// +optional
	Message string `json:"message,omitempty"`
	// A brief CamelCase message indicating details about why the policy is in this state. e.g. 'enforced'
	// +optional
	Reason string `json:"reason,omitempty"`

	State ResourceState `json:"state,omitempty"`

	Status            ComplianceMap `json:"status,omitempty"`
	PlacementPolicies []string      `json:"placementPolicies,omitempty"`
	PlacementBindings []string      `json:"placementBindings,omitempty"`
}

// PolicyList is a list of Policy resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:lister-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Policy `json:"items"`
}

// PolicyScope where the policy applies
type PolicyScope struct {
	Clusters  string   `json:"clusters,omitempty" protobuf:"bytes,5,rep,name=clusters"`
	Namespace []string `json:"namespace,omitempty" protobuf:"bytes,5,rep,name=namespace"`
}

// RoleTemplate describes how a role should look like
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:lister-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RoleTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	ComplianceType ComplianceType `json:"complianceType,omitempty"`

	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`

	// Rules holds all the PolicyRules for this Role
	Rules []PolicyRuleTemplate `json:"rules,omitempty" protobuf:"bytes,2,rep,name=rules"`

	Status TemplateStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

// GenericTemplate describes how an attribute should look like
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:lister-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GenericTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`

	// Rules holds all the attribute rules
	Rules []string `json:"rules,omitempty" protobuf:"bytes,2,rep,name=rules"`

	Status GenericTemplateStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
}

//TemplateStatus hold the status result
type TemplateStatus struct {
	ComplianceState ComplianceState `json:"Compliant,omitempty"` // Compliant, NonCompliant, UnkownCompliancy
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []Condition `json:"conditions,omitempty"`

	Validity Validity `json:"Validity,omitempty"` // a template can be invalid if it has conflicting roles
}

//GenericTemplateStatus hold the status result
type GenericTemplateStatus struct {
	ComplianceState ComplianceState `json:"Compliant"` // Compliant, NonCompliant, UnkownCompliancy
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Violations map[string]map[string][]string `json:"Violations,omitempty"` //first map for apiVersion.resource.namespace the third of the the instance of the resource and value is for violations per instance

	//Validity Validity `json:"Validity,omitempty"` // a template can be invalid if it has conflicting roles
}

//Validity describes if it is valid or not
type Validity struct {
	Valid  *bool  `json:"valid,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// RoleTemplateList lists roleTemplates
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:lister-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RoleTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []RoleTemplate `json:"items"`
}

// PolicyRuleTemplate holds information that describes a policy rule, but does not contain information
// about who the rule applies to or which namespace the rule applies to. We added the compliance type to it for HCM
type PolicyRuleTemplate struct {
	// ComplianceType specifies wether it is a : //musthave, mustnothave, mustonlyhave
	ComplianceType ComplianceType `json:"complianceType"`
	// PolicyRule
	PolicyRule rbacv1.PolicyRule `json:"policyRule"`
}
