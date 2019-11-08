/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strings"
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/ma-mcm-controller/pkg/apis/mcm/v1alpha1"
)

func TestTransformMutationPolicy(t *testing.T) {
	var m mcm.MutationPolicy
	UnmarshalFile("../../test-data/mutation_policy.json", &m, t)
	node := MutationPolicyResource{&m}.BuildNode()
	testUids := []string{"eb790c2e-361f-11e9-85ca-00163e019656", "3f4f13f2-f900-11e9-aa82-00163e01bcd9"}
	// Test only the fields that exist in MutationPolicy - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "MutationPolicy", t)
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("mutatedResources", node.Properties["mutatedResources"], 2, t)
	AssertEqual("severity", node.Properties["severity"], "", t)
	AssertEqual("remediationAction", node.Properties["remediationAction"], "inform", t)

	testMap := make(map[string]bool)
	actual := strings.Split(node.GetMetadata("_mutatedUIDs"), ",")
	AssertEqual("mutatedUIDLength", len(testUids), len(actual), t)
	//Put the actual values in a Map to  verify
	for _, actualItem := range actual {
		testMap[actualItem] = true
	}
	for _, testItem := range testUids {
		_, ok := testMap[testItem]
		AssertEqual(testItem, ok, true, t)
	}

}
