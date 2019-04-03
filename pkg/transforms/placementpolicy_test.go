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

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

func TestTransformPlacementPolicy(t *testing.T) {
	var p mcm.PlacementPolicy
	UnmarshalFile("../../test-data/placementpolicy.json", &p, t)
	node := transformPlacementPolicy(&p)

	// Test only the fields that exist in placementpolicy - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementPolicy", t)
	AssertEqual("replicas", node.Properties["replicas"], int64(1), t)
	AssertDeepEqual("decisions", node.Properties["decisions"], []string{"remote-kl"}, t)
}
