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

	v1 "k8s.io/api/core/v1"
)

func TestTransformPersistentVolume(t *testing.T) {
	var p v1.PersistentVolume
	UnmarshalFile("../../test-data/persistentvolume.json", &p, t)
	node := PersistentVolumeResource{&p}.BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("reclaimPolicy", node.Properties["reclaimPolicy"], "Delete", t)
	AssertEqual("status", node.Properties["status"], "Bound", t)
	AssertEqual("type", node.Properties["type"], "Hostpath", t)
	AssertEqual("capacity", node.Properties["capacity"], "5Gi", t)
	AssertDeepEqual("accessMode", node.Properties["accessMode"], []string{"ReadWriteOnce"}, t)
	AssertEqual("claimRef", node.Properties["claimRef"], "kube-system/test-pvc", t)
	AssertEqual("path", node.Properties["path"], "/var/lib/icp/helmrepo", t)
}
