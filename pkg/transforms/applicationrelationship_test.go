package transforms

import (
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

func TestTransformApplicationRelationship(t *testing.T) {
	var aR mcm.ApplicationRelationship
	UnmarshalFile("../../test-data/applicationrelationship.json", &aR, t)
	node := transformApplicationRelationship(&aR)

	// Test only the fields that exist in applicationrelationship - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "ApplicationRelationship", t)
	AssertEqual("destination", node.Properties["destination"], "test-test-redismaster", t)
	AssertEqual("source", node.Properties["source"], "test-test", t)
	AssertEqual("type", node.Properties["type"], "contains", t)
}
