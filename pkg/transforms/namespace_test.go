package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformNamespace(t *testing.T) {
	var n v1.Namespace
	UnmarshalFile("../../test-data/namespace.json", &n, t)
	node := transformNamespace(&n)

	// Test only the fields that exist in namespace - the common test will test the other bits
	AssertEqual("status", node.Properties["status"], "Active", t)
}
