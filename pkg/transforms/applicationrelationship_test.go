/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

func TestTransformApplicationRelationship(t *testing.T) {
	var aR mcm.ApplicationRelationship
	UnmarshalFile("../../test-data/applicationrelationship.json", &aR, t)
	node := ApplicationRelationshipResource{&aR}.BuildNode()

	// Test only the fields that exist in applicationrelationship - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "ApplicationRelationship", t)
	AssertEqual("destination", node.Properties["destination"], "test-test-redismaster", t)
	AssertEqual("source", node.Properties["source"], "test-test", t)
	AssertEqual("type", node.Properties["type"], "contains", t)
}
