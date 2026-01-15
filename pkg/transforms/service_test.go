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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformService(t *testing.T) {
	var s v1.Service
	UnmarshalFile("service.json", &s, t)
	node := ServiceResourceBuilder(&s, newUnstructuredService()).BuildNode()

	AssertEqual("kind", node.Properties["kind"], "Service", t)
	AssertDeepEqual("ips", node.Properties["ips"], []interface{}{"1.2.3.4", "2.3.4.5"}, t)
	AssertDeepEqual("servicePort", node.Properties["servicePort"], []interface{}{3333, 3334}, t)
	AssertDeepEqual("nodePort", node.Properties["nodePort"], []interface{}{30005, 30006}, t)
	AssertDeepEqual("targetPort", node.Properties["targetPort"], []interface{}{3333, 3334}, t)
	AssertDeepEqual("selector", node.Properties["selector"], map[string]interface{}{"app": "test-fixture-selector", "release": "test-fixture-selector-release"}, t)
}

func TestServiceBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := []Node{{
		UID: "local-cluster/uuid-fake-pod",
		Properties: map[string]interface{}{"kind": "Pod", "namespace": "default", "name": "fake-pod",
			"label": map[string]string{"app": "test-fixture-selector", "release": "test-fixture-selector-release"}},
	}}
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource cronjob.json
	var svc v1.Service
	UnmarshalFile("service.json", &svc, t)
	edges := ServiceResourceBuilder(&svc, newUnstructuredService()).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Service has no edges:", len(edges), 1, t)

	AssertEqual("Service usedBy: ", edges[0].DestKind, "Pod", t)
}

func newUnstructuredService() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"creationTimestamp": "2019-05-07T18:23:00Z",
			"labels": map[string]interface{}{
				"app":      "test-fixture",
				"chart":    "test-fixture-3.1.2",
				"heritage": "Tiller",
				"release":  "test-fixture",
			},
			"name":            "test-fixture-test-fixture",
			"namespace":       "default",
			"resourceVersion": "1234",
			"selfLink":        "/api/v1/namespaces/kube-system/services/test-fixture-test-fixture",
			"uid":             "255596bf-70f5-11e9-acdf-00163e03g660",
		},
		"spec": map[string]interface{}{
			"clusterIP":             "10.0.0.5",
			"externalTrafficPolicy": "Cluster",
			"ports": []interface{}{
				map[string]interface{}{
					"name":       "test-fixture",
					"nodePort":   30005,
					"port":       3333,
					"protocol":   "TCP",
					"targetPort": 3333,
				},
				map[string]interface{}{
					"name":       "test-fixture-2",
					"nodePort":   30006,
					"port":       3334,
					"protocol":   "TCP",
					"targetPort": 3334,
				},
			},
			"selector": map[string]interface{}{
				"app":     "test-fixture-selector",
				"release": "test-fixture-selector-release",
			},
			"sessionAffinity": "None",
			"type":            "NodePort",
		},
		"status": map[string]interface{}{
			"loadBalancer": map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"ip": "1.2.3.4",
					},
					map[string]interface{}{
						"ip": "2.3.4.5",
					},
				},
			},
		},
	},
	}
}
