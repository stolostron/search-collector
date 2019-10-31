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

	mcm "github.ibm.com/IBMPrivateCloud/ma-mcm-controller/pkg/apis/mcm/v1alpha1"
)

func TestTransformMutationPolicy(t *testing.T) {
	var m mcm.MutationPolicy
	UnmarshalFile("../../test-data/mutation_policy.json", &m, t)
	node := MutationPolicyResource{&m}.BuildNode()

	// Test only the fields that exist in MutationPolicy - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "MutationPolicy", t)
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("mutatedResources", node.Properties["mutatedResources"], 5, t)
	AssertEqual("severity", node.Properties["severity"], "", t)
	AssertEqual("remediationAction", node.Properties["remediationAction"], "inform", t)
}
