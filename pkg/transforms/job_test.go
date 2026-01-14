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

	v1 "k8s.io/api/batch/v1"
)

func TestTransformJob(t *testing.T) {
	var j v1.Job
	UnmarshalFile("job.json", &j, t)
	node := JobResourceBuilder(&j, newUnstructuredJob()).BuildNode()

	// Test only the fields that exist in job - the common test will test the other bits
	AssertEqual("active", node.Properties["active"], 1, t)
	AssertEqual("failed", node.Properties["failed"], 1, t)
	AssertEqual("successful", node.Properties["successful"], int64(1), t)
	AssertEqual("completions", node.Properties["completions"], int64(1), t)
	AssertEqual("parallelism", node.Properties["parallelism"], int64(1), t)
}

func TestJobBuildEdges(t *testing.T) {
	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := make([]Node, 0)
	nodeStore := BuildFakeNodeStore(nodes)

	// Build edges from mock resource job.json
	var j v1.Job
	UnmarshalFile("job.json", &j, t)
	edges := JobResourceBuilder(&j, newUnstructuredJob()).BuildEdges(nodeStore)

	// Validate results
	AssertEqual("Job has no edges:", len(edges), 0, t)
}

func newUnstructuredJob() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "batch",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"creationTimestamp": "2019-02-21T21:47:07Z",
			"labels": map[string]interface{}{
				"controller-uid": "3c0e82b5-3622-11e9-85ca-00163e019656",
				"job-name":       "fake-job",
			},
			"name":            "fake-job",
			"namespace":       "kube-system",
			"resourceVersion": "4356",
			"selfLink":        "/apis/batch/v1/namespaces/default/jobs/fake-job",
			"uid":             "3c0e82b5-3622-11e9-85ca-00163e019656",
		},
		"spec": map[string]interface{}{
			"backoffLimit": 6,
			"completions":  1,
			"parallelism":  1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"controller-uid": "3c0e82b5-3622-11e9-85ca-00163e019656",
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
					},
					"creationTimestamp": nil,
					"labels": map[string]interface{}{
						"controller-uid": "3c0e82b5-3622-11e9-85ca-00163e019656",
						"job-name":       "fake-job",
					},
					"name": "fake-job",
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"command": []interface{}{
								"python",
								"/app/scripts/onboard-script.py",
							},
							"envs": []interface{}{
								"env-1",
								"env-2",
							},
							"image":           "fake-image:3.1.2",
							"imagePullPolicy": "IfNotPresent",
							"name":            "fake-job",
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "500m",
									"memory": "256Mi",
								},
							},
							"terminationMessagePath":   "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"volumeMounts": []interface{}{
								map[string]interface{}{
									"mountPath": "/app/fake",
									"name":      "fake-json",
								},
							},
						},
						map[string]interface{}{
							"name":            "fake-job-2",
							"envs":            []interface{}{"env-3"},
							"image":           "fake-image:3.1.3",
							"imagePullPolicy": "IfNotPresent",
						},
					},
					"dnsPolicy":                     "ClusterFirst",
					"nodeSelector":                  map[string]interface{}{},
					"priorityClassName":             "system-cluster-critical",
					"restartPolicy":                 "OnFailure",
					"schedulerName":                 "default-scheduler",
					"securityContext":               map[string]interface{}{},
					"terminationGracePeriodSeconds": 30,
					"tolerations": []interface{}{
						map[string]interface{}{
							"effect":   "NoSchedule",
							"key":      "dedicated",
							"operator": "Exists",
						},
						map[string]interface{}{
							"key":      "CriticalAddonsOnly",
							"operator": "Exists",
						},
					},
					"volumes": []interface{}{
						map[string]interface{}{
							"configMap": map[string]interface{}{
								"defaultMode": 484,
								"name":        "fake-json",
							},
							"name": "fake-json",
						},
					},
				},
			},
		},
		"status": map[string]interface{}{
			"active":         1,
			"completionTime": "2019-02-21T21:47:45Z",
			"conditions": []interface{}{
				map[string]interface{}{
					"lastProbeTime":      "2019-02-21T21:47:45Z",
					"lastTransitionTime": "2019-02-21T21:47:45Z",
					"status":             "True",
					"type":               "Complete",
				},
			},
			"failed":    1,
			"startTime": "2019-02-21T21:47:07Z",
			"succeeded": 1,
		},
	},
	}
}
