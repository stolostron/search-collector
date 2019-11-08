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

func TestTransformPlacementBinding(t *testing.T) {
	var p mcm.PlacementBinding
	UnmarshalFile("../../test-data/placementbinding.json", &p, t)
	node := PlacementBindingResource{&p}.BuildNode()

	// Test only the fields that exist in placementbinding - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementBinding", t)
	AssertEqual("placementpolicy", node.Properties["placementpolicy"], "foo-test (PlacementPolicy)", t)
	AssertDeepEqual("subject", node.Properties["subject"], []string{"foo-test (Deployable)"}, t)
}
