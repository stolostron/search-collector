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
	"testing"

	policy "github.com/stolostron/governance-policy-propagator/api/v1"
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
}

func TestTransformConfigPolicy(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("configurationpolicy.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	configResource := StandalonePolicyResourceBuilder(unstructured)

	node := configResource.BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("remediationAction", node.Properties["remediationAction"], "inform", t)
	AssertEqual("severity", node.Properties["severity"], "low", t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], true, t)
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
}

func TestTransformCertPolicy(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("certpolicy.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	certResource := StandalonePolicyResourceBuilder(unstructured)

	node := certResource.BuildNode()

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("compliant", node.Properties["compliant"], "Compliant", t)
	AssertEqual("severity", node.Properties["severity"], "low", t)
	AssertEqual("disabled", node.Properties["disabled"], false, t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], true, t)
}
