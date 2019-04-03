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

	com "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/compliance/v1alpha1"
)

func TestTransformCompliance(t *testing.T) {
	var c com.Compliance
	UnmarshalFile("../../test-data/compliance.json", &c, t)
	node := transformCompliance(&c)

	// Test only the fields that exist in compliance - the common test will test the other bits
	AssertEqual("policyCompliant", node.Properties["policyCompliant"], int64(1), t)
	AssertEqual("policyTotal", node.Properties["policyTotal"], int64(1), t)
	AssertEqual("clusterCompliant", node.Properties["clusterCompliant"], int64(1), t)
	AssertEqual("clusterTotal", node.Properties["clusterTotal"], int64(1), t)
	AssertEqual("status", node.Properties["status"], "compliant", t)
}
