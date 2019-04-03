/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"
)

func TestTransformPolicy(t *testing.T) {
	var p mcm.Policy
	UnmarshalFile("../../test-data/policy.json", &p, t)
	node := transformPolicy(&p)

	// Test only the fields that exist in policy - the common test will test the other bits
	AssertEqual("remediationAction", node.Properties["remediationAction"], "enforce", t)
	AssertEqual("compliant", node.Properties["compliant"], "Compliant", t)
	AssertEqual("valid", node.Properties["valid"], true, t)
	AssertEqual("numRules", node.Properties["numRules"], int64(1), t)
}
