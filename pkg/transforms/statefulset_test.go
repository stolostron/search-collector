package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformStatefulSet(t *testing.T) {
	var s v1.StatefulSet
	UnmarshalFile("../../test-data/statefulset.json", &s, t)
	node := TransformStatefulSet(&s)

	// Test only the fields that exist in stateful set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int32(1), t)
	AssertEqual("desired", node.Properties["desired"], int32(1), t)
}
