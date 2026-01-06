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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformPod(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod.json", &p, t)
	node := PodResourceBuilder(&p, newUnstructuredPod()).BuildNode()

	// Build time struct matching time in test data
	date := time.Date(2019, 02, 21, 21, 30, 33, 0, time.UTC)

	// Test only the fields that exist in pods - the common test will test the other bits

	AssertEqual("kind", node.Properties["kind"], "Pod", t)
	AssertEqual("hostIP", node.Properties["hostIP"], "1.1.1.1", t)
	AssertEqual("podIP", node.Properties["podIP"], "2.2.2.2", t)
	AssertEqual("restarts", node.Properties["restarts"], int64(0), t)
	AssertDeepEqual("container", node.Properties["container"], []string{"fake-pod"}, t)
	AssertDeepEqual("image", node.Properties["image"], []string{"fake-image:latest"}, t)
	AssertDeepEqual("initContainers", node.Properties["initContainers"], []string{"init-container-1", "init-container-2"}, t)
	AssertEqual("startedAt", node.Properties["startedAt"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("status", node.Properties["status"], string(v1.PodRunning), t)
	AssertEqual("_ownerUID", node.Properties["_ownerUID"], "local-cluster/eb762405-361f-11e9-85ca-00163e019656", t)
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"ContainersReady":           "True",
		"Initialized":               "True",
		"PodReadyToStartContainers": "True",
		"PodScheduled":              "True",
		"Ready":                     "True",
	}, t)
}

func TestTransformPodInitWaiting(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod-init-waiting.json", &p, t)
	node := PodResourceBuilder(&p, newUnstructuredPod()).BuildNode()

	AssertEqual("podIP", node.Properties["podIP"], "2.2.2.3", t)
	AssertEqual("restarts", node.Properties["restarts"], int64(2), t)
	AssertEqual("status", node.Properties["status"], "Init:CrashLoopBackOff", t)
}

func TestTransformPodInitFailed(t *testing.T) {
	var p v1.Pod
	UnmarshalFile("pod-init-failed.json", &p, t)
	node := PodResourceBuilder(&p, newUnstructuredPod()).BuildNode()

	// Test only status of pood with a completed init container
	AssertEqual("status", node.Properties["status"], "Init:ExitCode:255", t)
}

func TestPodBuildEdges(t *testing.T) {

	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := []Node{{
		UID:        "uuid-123-secret",
		Properties: map[string]interface{}{"kind": "Secret", "namespace": "default", "name": "test-secret"},
	}, {
		UID:        "uuid-123-configmap",
		Properties: map[string]interface{}{"kind": "ConfigMap", "namespace": "default", "name": "test-configmap"},
	}, {
		UID:        "uuid-123-pv",
		Properties: map[string]interface{}{"kind": "PersistentVolume", "namespace": "_NONE", "name": "test-pv"},
	}, {
		UID:        "uuid-123-pvc",
		Properties: map[string]interface{}{"kind": "PersistentVolumeClaim", "namespace": "default", "name": "test-pvc", "volumeName": "test-pv"},
	}, {
		UID:        "uuid-123-node",
		Properties: map[string]interface{}{"kind": "Node", "namespace": "_NONE", "name": "1.1.1.1"},
	}, {
		UID:        "local-cluster/uuid-fake-pod-aaaaa",
		Properties: map[string]interface{}{"kind": "Pod", "namespace": "default", "name": "fake-pod-aaaa"},
	}}
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource pod.json
	var p v1.Pod
	UnmarshalFile("pod.json", &p, t)
	edges := PodResourceBuilder(&p, newUnstructuredPod()).BuildEdges(nodeStore)

	// Verify created edges.
	AssertEqual("Pod edge total: ", len(edges), 5, t)
	AssertEqual("Pod attachedTo", edges[0].DestKind, "Secret", t)
	AssertEqual("Pod attachedTo", edges[1].DestKind, "ConfigMap", t)
	AssertEqual("Pod attachedTo", edges[2].DestKind, "PersistentVolumeClaim", t)
	AssertEqual("Pod attachedTo", edges[3].DestKind, "PersistentVolume", t)
	AssertEqual("Pod runsOn", edges[4].DestKind, "Node", t)
}

func newUnstructuredPod() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
				map[string]interface{}{
					"type":   "ContainersReady",
					"status": "True",
				},
				map[string]interface{}{
					"type":   "PodScheduled",
					"status": "True",
				},
				map[string]interface{}{
					"type":   "Initialized",
					"status": "True",
				},
				map[string]interface{}{
					"type":   "PodReadyToStartContainers",
					"status": "True",
				},
			},
		},
	}}
}
