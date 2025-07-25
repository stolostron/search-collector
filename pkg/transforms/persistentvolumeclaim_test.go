/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformPersistentVolumeClaim(t *testing.T) {
	var p v1.PersistentVolumeClaim
	UnmarshalFile("persistentvolumeclaim.json", &p, t)
	node := PersistentVolumeClaimResourceBuilder(&p, newUnstructuredPersistentVolumeClaim()).BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("volumeName", node.Properties["volumeName"], "test-pv", t)
	AssertEqual("volumeMode", node.Properties["volumeMode"], "Filesystem", t)
	AssertEqual("status", node.Properties["status"], "Bound", t)
	AssertEqual("storageClassName", node.Properties["storageClassName"], "test-storage", t)
	AssertEqual("capacity", node.Properties["capacity"], "5Gi", t)
	AssertEqual("requestedStorage", node.Properties["requestedStorage"], int64(5368709120), t) // 5Gi
	AssertDeepEqual("accessMode", node.Properties["accessMode"], []string{"ReadWriteOnce"}, t)
}

func newUnstructuredPersistentVolumeClaim() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "PersistentVolumeClaim",
		"spec": map[string]interface{}{
			"volumeMode": "Filesystem",
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"storage": "5Gi",
				},
			},
		},
	}}
}
