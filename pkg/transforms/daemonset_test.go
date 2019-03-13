package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformDaemonSet(t *testing.T) {
	var d v1.DaemonSet
	UnmarshalFile("../../test-data/daemonset.json", &d, t)
	node := transformDaemonSet(&d)

	// Test only the fields that exist in daemonset - the common test will test the other bits
	AssertEqual("available", node.Properties["available"], int32(1), t)
	AssertEqual("current", node.Properties["current"], int32(1), t)
	AssertEqual("desired", node.Properties["desired"], int32(1), t)
	AssertEqual("ready", node.Properties["ready"], int32(1), t)
	AssertEqual("updated", node.Properties["updated"], int32(1), t)
}
