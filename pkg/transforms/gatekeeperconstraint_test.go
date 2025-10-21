// Copyright (c) 2025 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformGKConstraintNonCompliant(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("gk-constraint-livenessprobe.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	constraintResource := GkConstraintResourceBuilder(unstructured)

	node := constraintResource.BuildNode()

	// Only test the fields specific to Gatekeeper Constraints
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], false, t)
	obj1 := relatedObject{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "fake-deployment",
	}
	obj2 := relatedObject{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "multicluster-engine",
		Name:      "provider-credential-controller",
	}
	AssertEqual("relObjs", node.GetMetadata("relObjs"),
		"["+obj1.String()+" "+obj2.String()+"]", t)
}

func TestTransformGKConstraintCompliant(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("gk-constraint-requiredlabels.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	constraintResource := GkConstraintResourceBuilder(unstructured)

	node := constraintResource.BuildNode()

	// Only test the fields specific to Gatekeeper Constraints
	AssertEqual("compliant", node.Properties["compliant"], "Compliant", t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], false, t)
	AssertEqual("relObjs", node.GetMetadata("relObjs"), "", t)
}
