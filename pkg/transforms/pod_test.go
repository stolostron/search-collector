/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
)

func TestTransformPod(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod.json", &p, t)
	node := PodResourceBuilder(&p).BuildNode()

	// Build time struct matching time in test data
	date := time.Date(2019, 02, 21, 21, 30, 33, 0, time.UTC)

	// Test only the fields that exist in pods - the common test will test the other bits

	AssertEqual("kind", node.Properties["kind"], "Pod", t)
	AssertEqual("hostIP", node.Properties["hostIP"], "1.1.1.1", t)
	AssertEqual("podIP", node.Properties["podIP"], "2.2.2.2", t)
	AssertEqual("restarts", node.Properties["restarts"], int64(0), t)
	AssertDeepEqual("container", node.Properties["container"], []string{"fake-pod"}, t)
	AssertDeepEqual("image", node.Properties["image"], []string{"fake-image:latest"}, t)
	AssertEqual("startedAt", node.Properties["startedAt"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("status", node.Properties["status"], string(v1.PodRunning), t)
}

func TestTransformPodInitWaiting(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod-init-waiting.json", &p, t)
	node := PodResourceBuilder(&p).BuildNode()

	AssertEqual("podIP", node.Properties["podIP"], "2.2.2.3", t)
	AssertEqual("restarts", node.Properties["restarts"], int64(2), t)
	AssertEqual("status", node.Properties["status"], "Init:CrashLoopBackOff", t)
}

func TestTransformPodInitFailed(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod-init-failed.json", &p, t)
	node := PodResourceBuilder(&p).BuildNode()

	// Test only status of pood with a completed init container
	AssertEqual("status", node.Properties["status"], "Init:ExitCode:255", t)
}

func TestPodBuildEdges(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod.json", &p, t)

	byUID := make(map[string]Node)
	byKindNameNamespace := make(map[string]map[string]map[string]Node)
	n := Node{
		UID:        "uuid-123-secret",
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]string),
	}
	byUID["uuid-123-secret"] = n
	byKindNameNamespace["Secret"] = make(map[string]map[string]Node)
	byKindNameNamespace["Secret"]["default"] = make(map[string]Node)
	byKindNameNamespace["Secret"]["default"]["test-secret"] = n

	n_configmap := Node{
		UID:        "uuid-123-configmap",
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]string),
	}
	n_configmap.Properties["name"] = "test-configmap"
	byUID["uuid-123-configmap"] = n_configmap
	byKindNameNamespace["ConfigMap"] = make(map[string]map[string]Node)
	byKindNameNamespace["ConfigMap"]["default"] = make(map[string]Node)
	byKindNameNamespace["ConfigMap"]["default"]["test-configmap"] = n_configmap

	n_pvc := Node{
		UID:        "uuid-123-pvc",
		Properties: make(map[string]interface{}),
		Metadata:   make(map[string]string),
	}
	n_pvc.Properties["name"] = "test-pvc"
	byUID["uuid-123-pvc"] = n_configmap
	byKindNameNamespace["PersistentVolumeClaim"] = make(map[string]map[string]Node)
	byKindNameNamespace["PersistentVolumeClaim"]["default"] = make(map[string]Node)
	byKindNameNamespace["PersistentVolumeClaim"]["default"]["test-pvc"] = n_pvc

	store := NodeStore{
		ByUID:               byUID,
		ByKindNamespaceName: byKindNameNamespace,
	}

	edges := PodResourceBuilder(&p).BuildEdges(store)

	AssertEqual("Pod attachedTo Secret:", len(edges), 1, t)
}
