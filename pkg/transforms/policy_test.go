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

	policy "github.com/stolostron/governance-policy-propagator/pkg/apis/policy/v1"
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
