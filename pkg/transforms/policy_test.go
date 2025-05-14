/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"reflect"
	"testing"

	policy "github.com/stolostron/governance-policy-propagator/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformPolicy(t *testing.T) {
	var p policy.Policy
	UnmarshalFile("policy.json", &p, t)
	node := PolicyResourceBuilder(&p).BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("remediationAction", node.Properties["remediationAction"], "enforce", t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("numRules", node.Properties["numRules"], 1, t)
	assert.Len(t, node.Properties["annotation"], 3, "expected 3 annotations on the policy")
}

func TestTransformConfigPolicy(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("configurationpolicy.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	configResource := ConfigPolicyResourceBuilder(unstructured)

	node := configResource.BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("remediationAction", node.Properties["remediationAction"], "inform", t)
	AssertEqual("severity", node.Properties["severity"], "low", t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], true, t)
	obj1 := `{"v":"v1","k":"Namespace","n":"default"}`
	obj2 := `{"v":"v1","k":"Namespace","n":"nonexistent"}`
	AssertEqual("relObjs", node.GetMetadata("relObjs"),
		"["+obj1+" "+obj2+"]", t)
}

func TestTransformOperatorPolicy(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("operatorpolicy.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	operatorResource := OperatorPolicyResourceBuilder(unstructured)

	node := operatorResource.BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("remediationAction", node.Properties["remediationAction"], "inform", t)
	AssertEqual("severity", node.Properties["severity"], "critical", t)
	AssertEqual("deploymentAvailable", node.Properties["deploymentAvailable"], false, t)
	AssertEqual("upgradeAvailable", node.Properties["upgradeAvailable"], true, t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], false, t)

	objs := []relatedObject{{
		Group:     "operators.coreos.com",
		Version:   "v1alpha1",
		Kind:      "CatalogSource",
		Namespace: "openshift-marketplace",
		Name:      "redhat-operators",
		EdgeType:  compliantEdge,
	}, {
		Group:     "operators.coreos.com",
		Version:   "v1alpha1",
		Kind:      "ClusterServiceVersion",
		Namespace: "open-cluster-management",
		Name:      "advanced-cluster-management.v2.9.0",
		EdgeType:  noncompliantEdge,
	}}

	if !reflect.DeepEqual(node.Metadata["relObjs"], objs) {
		t.Errorf("relObjs EXPECTED: %T %v\n", node.Metadata["relObjs"], node.Metadata["relObjs"])
		t.Errorf("relObjs ACTUAL: %T %v\n", objs, objs)
		t.Fail()
	}
}

func TestTransformCertPolicy(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("certpolicy.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	certResource := CertPolicyResourceBuilder(unstructured)

	node := certResource.BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("severity", node.Properties["severity"], "low", t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], true, t)
	obj := `{"v":"v1","k":"Secret","ns":"default","n":"sample-secret"}`
	AssertEqual("relObjs", node.GetMetadata("relObjs"), "["+obj+"]", t)
}
