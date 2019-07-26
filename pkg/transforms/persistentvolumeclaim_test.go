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

func TestTransformPersistentVolumeClaim(t *testing.T) {
	var p v1.PersistentVolumeClaim
	UnmarshalFile("../../test-data/persistentvolumeclaim.json", &p, t)
	node := PersistentVolumeClaimResource{&p}.BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("volumeName", node.Properties["volumeName"], "test-pv", t)
	AssertEqual("status", node.Properties["status"], "Bound", t)
	AssertEqual("storageClassName", node.Properties["storageClassName"], "test-storage", t)
	AssertEqual("capacity", node.Properties["capacity"], "5Gi", t)
	AssertDeepEqual("accessMode", node.Properties["accessMode"], []string{"ReadWriteOnce"}, t)
}
