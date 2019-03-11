package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformReplicaSet(t *testing.T) {
	var r v1.ReplicaSet
	UnmarshalFile("../../test-data/replicaset.json", &r, t)
	node := TransformReplicaSet(&r)

	// Test only the fields that exist in replica set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int32(1), t)
	AssertEqual("desired", node.Properties["desired"], int32(1), t)
}
