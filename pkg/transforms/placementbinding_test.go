package transforms

import (
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

func TestTransformPlacementBinding(t *testing.T) {
	var p mcm.PlacementBinding
	UnmarshalFile("../../test-data/placementbinding.json", &p, t)
	node := transformPlacementBinding(&p)

	// Test only the fields that exist in placementbinding - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementBinding", t)
	AssertEqual("placementpolicy", node.Properties["placementpolicy"], "foo-test (PlacementPolicy)", t)
	AssertDeepEqual("subject", node.Properties["subject"], []string{"foo-test (Deployable)"}, t)
}
