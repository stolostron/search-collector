package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformDeployment(t *testing.T) {
	var d v1.Deployment
	UnmarshalFile("../../test-data/deployment.json", &d, t)
	node := transformDeployment(&d)

	// Test only the fields that exist in deployment - the common test will test the other bits
	AssertEqual("available", node.Properties["available"], int32(1), t)
	AssertEqual("current", node.Properties["current"], int32(1), t)
	AssertEqual("desired", node.Properties["desired"], int32(1), t)
	AssertEqual("ready", node.Properties["ready"], int32(1), t)
}
