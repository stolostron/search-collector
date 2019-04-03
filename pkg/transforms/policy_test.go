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
