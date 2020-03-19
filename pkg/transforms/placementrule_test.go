/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	app "github.com/IBM/multicloud-operators-placementrule/pkg/apis/app/v1alpha1"
)

func TestTransformPlacementRule(t *testing.T) {
	var p app.PlacementRule
	UnmarshalFile("../../test-data/placementrule.json", &p, t)
	node := PlacementRuleResource{&p}.BuildNode()

	// Test only the fields that exist in placementrule - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementRule", t)
	AssertEqual("apigroup", node.Properties["apigroup"], "apps.open-cluster-management.io", t)
}
