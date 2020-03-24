// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeBuilder is used to build the scheme for the seed types
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds to the SchemeBuilder
	AddToScheme = SchemeBuilder.AddToScheme
)

// GroupName is the group name used in this package.
const GroupName = "policy.mcm.ibm.com"

// SchemeGroupVersion is the group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Policy{}, &PolicyList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
