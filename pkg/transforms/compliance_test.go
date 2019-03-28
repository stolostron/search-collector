package transforms

import (
	"testing"

	com "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/compliance/v1alpha1"
)

func TestTransformCompliance(t *testing.T) {
	var c com.Compliance
	UnmarshalFile("../../test-data/compliance.json", &c, t)
	node := transformCompliance(&c)

	// Test only the fields that exist in compliance - the common test will test the other bits
	AssertEqual("policyCompliant", node.Properties["policyCompliant"], 1, t)
	AssertEqual("policyTotal", node.Properties["policyTotal"], 1, t)
	AssertEqual("clusterCompliant", node.Properties["clusterCompliant"], 1, t)
	AssertEqual("clusterTotal", node.Properties["clusterTotal"], 1, t)
	AssertEqual("status", node.Properties["status"], "compliant", t)
}
