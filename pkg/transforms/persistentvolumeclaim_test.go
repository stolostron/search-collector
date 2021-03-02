/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformPersistentVolumeClaim(t *testing.T) {
	var p v1.PersistentVolumeClaim
	UnmarshalFile("persistentvolumeclaim.json", &p, t)
	node := PersistentVolumeClaimResourceBuilder(&p).BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("volumeName", node.Properties["volumeName"], "test-pv", t)
	AssertEqual("status", node.Properties["status"], "Bound", t)
	AssertEqual("storageClassName", node.Properties["storageClassName"], "test-storage", t)
	AssertEqual("capacity", node.Properties["capacity"], "5Gi", t)
	AssertDeepEqual("accessMode", node.Properties["accessMode"], []string{"ReadWriteOnce"}, t)
}
