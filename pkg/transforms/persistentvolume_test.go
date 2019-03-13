package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformPersistentVolume(t *testing.T) {
	var p v1.PersistentVolume
	UnmarshalFile("../../test-data/persistentvolume.json", &p, t)
	node := transformPersistentVolume(&p)

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("reclaimPolicy", node.Properties["reclaimPolicy"], "Delete", t)
	AssertEqual("status", node.Properties["status"], "Bound", t)
	AssertEqual("type", node.Properties["type"], "Hostpath", t)
	AssertEqual("capacity", node.Properties["capacity"], int64(5368709120), t)
	AssertEqual("accessModes", node.Properties["accessModes"], "ReadWriteOnce", t)
	AssertEqual("claimRef", node.Properties["claimRef"], "kube-system/test-pvc", t)
	AssertEqual("path", node.Properties["path"], "/var/lib/icp/helmrepo", t)
}
