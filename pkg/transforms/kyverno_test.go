// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformKyvernoPolicy(t *testing.T) {
	p := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "Policy",
			"metadata": map[string]interface{}{
				"name":      "my-policy",
				"namespace": "my-app",
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
			},
			"spec": map[string]interface{}{
				"validationFailureAction": "Enforce",
				"random":                  "value",
			},
		},
	}
	rv := KyvernoPolicyResourceBuilder(&p)
	node := rv.node

	AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], "Enforce", t)
	AssertEqual("background", node.Properties["background"], true, t)
	AssertEqual("admission", node.Properties["admission"], true, t)
	AssertEqual("severity", node.Properties["severity"], "medium", t)

	// Check the default value for spec.validationFailureAction
	unstructured.RemoveNestedField(p.Object, "spec", "validationFailureAction")

	rv = KyvernoPolicyResourceBuilder(&p)
	node = rv.node
	AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], "Audit", t)
}

func TestTransformKyvernoClusterPolicy(t *testing.T) {
	p := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "my-policy",
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "critical",
				},
			},
			"spec": map[string]interface{}{
				"validationFailureAction": "Enforce",
				"random":                  "value",
				"background":              false,
				"admission":               false,
			},
		},
	}
	rv := KyvernoPolicyResourceBuilder(&p)
	node := rv.node

	AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], "Enforce", t)
	AssertEqual("background", node.Properties["background"], false, t)
	AssertEqual("admission", node.Properties["admission"], false, t)
	AssertEqual("severity", node.Properties["severity"], "critical", t)
}

func TestTransformKyvernoFailureAction(t *testing.T) {
	tests := []struct {
		clusterPolicy unstructured.Unstructured
		expected      string
		testName      string
	}{
		{
			clusterPolicy: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kyverno.io/v1",
					"kind":       "ClusterPolicy",
					"metadata": map[string]interface{}{
						"name": "my-policy",
						"annotations": map[string]interface{}{
							"policies.kyverno.io/severity": "critical",
						},
					},
					"spec": map[string]interface{}{
						"validationFailureAction": "Enforce",
						"random":                  "value",
						"background":              false,
						"admission":               false,
						"rules": []interface{}{
							map[string]interface{}{"validate": map[string]interface{}{
								"failureAction": "Audit",
							}},
							map[string]interface{}{"validate": map[string]interface{}{
								"failureAction": "Enforce",
							}},
						},
					},
				},
			},
			expected: "Audit/Enforce",
			testName: "Test mixed failureActions",
		},
		{
			clusterPolicy: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kyverno.io/v1",
					"kind":       "ClusterPolicy",
					"metadata": map[string]interface{}{
						"name": "my-policy",
						"annotations": map[string]interface{}{
							"policies.kyverno.io/severity": "critical",
						},
					},
					"spec": map[string]interface{}{
						"validationFailureAction": "Enforce",
						"random":                  "value",
						"background":              false,
						"admission":               false,
						"rules": []interface{}{
							map[string]interface{}{"validate": map[string]interface{}{
								"failureAction": "Audit",
							}},
							map[string]interface{}{"validate": map[string]interface{}{
								"failureAction": "Audit",
							}},
						},
					},
				},
			},
			expected: "Audit",
			testName: "Test identical failureActions",
		},
		{
			clusterPolicy: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kyverno.io/v1",
					"kind":       "ClusterPolicy",
					"metadata": map[string]interface{}{
						"name": "my-policy",
						"annotations": map[string]interface{}{
							"policies.kyverno.io/severity": "critical",
						},
					},
					"spec": map[string]interface{}{
						"random":     "value",
						"background": false,
						"admission":  false,
						"rules": []interface{}{
							map[string]interface{}{"generate": map[string]interface{}{}},
						},
					},
				},
			},
			expected: "Audit",
			testName: "Test generate rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			rv := KyvernoPolicyResourceBuilder(&tt.clusterPolicy)
			node := rv.node

			AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], tt.expected, t)
		})
	}
}

func TestTransformKyvernoValidationAction(t *testing.T) {
	validatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "ValidatingPolicy",
			"metadata": map[string]interface{}{
				"name": "require-kubecost-labels",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "272ec5b8-892b-40da-8b92-af141c377daa",
			},
			"spec": map[string]interface{}{
				"validationActions": []interface{}{"Deny"},
				"validations": []interface{}{
					map[string]interface{}{
						"expression": "'owner' in object.metadata.labels",
						"message":    "The owner label is required",
					},
				},
			},
		},
	}

	mutatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "MutatingPolicy",
			"metadata": map[string]interface{}{
				"name": "add-app-labels",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "medium",
				},
				"uid": "373ec5b8-892b-40da-8b92-af141c377dbb",
			},
			"spec": map[string]interface{}{
				"matchConstraints": map[string]interface{}{
					"resourceRules": []interface{}{
						map[string]interface{}{
							"apiGroups": []interface{}{""},
							"resources": []interface{}{"pods"},
						},
					},
				},
				"mutations": []interface{}{
					map[string]interface{}{
						"patchType": "ApplyConfiguration",
						"applyConfiguration": map[string]interface{}{
							"expression": "Object{metadata: Object.metadata{labels: Object.metadata.labels{app: \"default\"}}}",
						},
					},
				},
			},
		},
	}

	imageValidatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "ImageValidatingPolicy",
			"metadata": map[string]interface{}{
				"name": "verify-image-signature",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "high",
				},
				"uid": "474ec5b8-892b-40da-8b92-af141c377dcc",
			},
			"spec": map[string]interface{}{
				"validationActions": []interface{}{"Audit", "Deny"},
				"attestors": []interface{}{
					map[string]interface{}{
						"name": "cosign-key",
						"cosign": map[string]interface{}{
							"key": map[string]interface{}{
								"value": "-----BEGIN PUBLIC KEY-----\ntest\n-----END PUBLIC KEY-----",
							},
						},
					},
				},
				"validations": []interface{}{
					map[string]interface{}{
						"expression": "true",
						"message":    "Image validation passed",
					},
				},
			},
		},
	}

	generatingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "GeneratingPolicy",
			"metadata": map[string]interface{}{
				"name": "generate-network-policy",
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "kyverno",
				},
				"annotations": map[string]interface{}{
					"policies.kyverno.io/severity": "low",
				},
				"uid": "575ec5b8-892b-40da-8b92-af141c377ddd",
			},
			"spec": map[string]interface{}{
				"generate": []interface{}{
					map[string]interface{}{
						"apiVersion": "networking.k8s.io/v1",
						"kind":       "NetworkPolicy",
						"name":       "test-policy",
					},
				},
				"evaluation": map[string]interface{}{
					"synchronize": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
	}

	validatingPolicyDisabled := validatingPolicy.DeepCopy()
	spec := validatingPolicyDisabled.Object["spec"].(map[string]interface{})
	spec["validationActions"] = []interface{}{"Audit"}
	spec["evaluation"] = map[string]interface{}{
		"background": map[string]interface{}{"enabled": false},
		"admission":  map[string]interface{}{"enabled": false},
	}

	imageValidatingPolicyDisabled := imageValidatingPolicy.DeepCopy()
	imageSpec := imageValidatingPolicyDisabled.Object["spec"].(map[string]interface{})
	imageSpec["evaluation"] = map[string]interface{}{
		"background": map[string]interface{}{"enabled": false},
		"admission":  map[string]interface{}{"enabled": false},
	}

	namespacedValidatingPolicy := validatingPolicy.DeepCopy()
	namespacedValidatingPolicy.Object["kind"] = "NamespacedValidatingPolicy"
	namespacedValidatingPolicy.Object["metadata"].(map[string]interface{})["namespace"] = "test-namespace"

	tests := []struct {
		testName           string
		policy             *unstructured.Unstructured
		expectedValidation string
		expectedBackground bool
		expectedAdmission  bool
		expectedSeverity   string
	}{
		{
			testName:           "Test ValidatingPolicy with Deny action",
			policy:             validatingPolicy,
			expectedValidation: "Deny",
			expectedBackground: true,
			expectedAdmission:  true,
			expectedSeverity:   "medium",
		},
		{
			testName:           "Test MutatingPolicy",
			policy:             mutatingPolicy,
			expectedValidation: "",
			expectedBackground: true,
			expectedAdmission:  true,
			expectedSeverity:   "medium",
		},
		{
			testName:           "Test ImageValidatingPolicy",
			policy:             imageValidatingPolicy,
			expectedValidation: "Audit/Deny",
			expectedBackground: true,
			expectedAdmission:  true,
			expectedSeverity:   "high",
		},
		{
			testName:           "Test GeneratingPolicy",
			policy:             generatingPolicy,
			expectedValidation: "",
			expectedBackground: true,
			expectedAdmission:  true,
			expectedSeverity:   "low",
		},
		{
			testName:           "Test ValidatingPolicy with evaluation disabled",
			policy:             validatingPolicyDisabled,
			expectedValidation: "Audit",
			expectedBackground: false,
			expectedAdmission:  false,
			expectedSeverity:   "medium",
		},
		{
			testName:           "Test ImageValidatingPolicy with evaluation disabled",
			policy:             imageValidatingPolicyDisabled,
			expectedValidation: "Audit/Deny",
			expectedBackground: false,
			expectedAdmission:  false,
			expectedSeverity:   "high",
		},
		{
			testName:           "Test NamespacedValidatingPolicy",
			policy:             namespacedValidatingPolicy,
			expectedValidation: "Deny",
			expectedBackground: true,
			expectedAdmission:  true,
			expectedSeverity:   "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			rv := KyvernoPolicyResourceBuilder(tt.policy)
			node := rv.node

			AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], tt.expectedValidation, t)
			AssertEqual("background", node.Properties["background"], tt.expectedBackground, t)
			AssertEqual("admission", node.Properties["admission"], tt.expectedAdmission, t)
			AssertEqual("severity", node.Properties["severity"], tt.expectedSeverity, t)
		})
	}
}

func TestKyvernoPolicyEdges(t *testing.T) {
	// Establish the config
	config.InitConfig()

	configmap := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "zk-kafka-address",
			"uid":       "18b016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test2",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "zk-kafka-address",
				"generate.kyverno.io/policy-namespace":  "",
				"generate.kyverno.io/rule-name":         "k-kafka-address",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Namespace",
				"generate.kyverno.io/trigger-namespace": "",
				"generate.kyverno.io/trigger-uid":       "12345",
				"generate.kyverno.io/trigger-version":   "v1",
				"somekey":                               "somevalue",
			},
		},
	}}

	configmapTwo := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "hello-cofigmap",
			"uid":       "777016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test2",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "kyverno-policy-test",
				"generate.kyverno.io/policy-namespace":  "test2",
				"generate.kyverno.io/rule-name":         "k-kafka-address",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Namespace",
				"generate.kyverno.io/trigger-namespace": "",
			},
		},
	}}

	configmapThree := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "generated-secret",
			"uid":       "888016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test4",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "generate-secret-policy",
				"generate.kyverno.io/policy-namespace":  "",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Namespace",
				"generate.kyverno.io/trigger-namespace": "",
			},
		},
	}}

	configmapFour := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "generated-config",
			"uid":       "999016fe-1931-4e80-95d1-d51d3b936e24",
			"namespace": "test3",
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by":          "kyverno",
				"generate.kyverno.io/policy-name":       "generate-configmap-policy",
				"generate.kyverno.io/policy-namespace":  "test4",
				"generate.kyverno.io/trigger-group":     "",
				"generate.kyverno.io/trigger-kind":      "Pod",
				"generate.kyverno.io/trigger-namespace": "test4",
			},
		},
	}}

	kyvernoPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "Policy",
			"metadata": map[string]interface{}{
				"name":      "kyverno-policy-test",
				"namespace": "test2",
				"uid":       "777738fa-0591-44da-8a06-98ea7d74a7f7",
			},
		},
	}

	kyvernoClusterpolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "zk-kafka-address",
				"uid":  "8fc338fa-0591-44da-8a06-98ea7d74a7f7",
			},
		},
	}

	kyvernoGeneratingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "GeneratingPolicy",
			"metadata": map[string]interface{}{
				"name": "generate-secret-policy",
				"uid":  "9fc338fa-0591-44da-8a06-98ea7d74a7f8",
			},
		},
	}

	kyvernoNamespacedGeneratingPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "policies.kyverno.io/v1",
			"kind":       "NamespacedGeneratingPolicy",
			"metadata": map[string]interface{}{
				"name":      "generate-configmap-policy",
				"namespace": "test4",
				"uid":       "afc338fa-0591-44da-8a06-98ea7d74a7f9",
			},
		},
	}

	nodes := []Node{
		KyvernoPolicyResourceBuilder(kyvernoPolicy).BuildNode(),
		KyvernoPolicyResourceBuilder(kyvernoClusterpolicy).BuildNode(),
		KyvernoPolicyResourceBuilder(kyvernoGeneratingPolicy).BuildNode(),
		KyvernoPolicyResourceBuilder(kyvernoNamespacedGeneratingPolicy).BuildNode(),
		GenericResourceBuilder(configmap).BuildNode(),
		GenericResourceBuilder(configmapTwo).BuildNode(),
		GenericResourceBuilder(configmapThree).BuildNode(),
		GenericResourceBuilder(configmapFour).BuildNode(),
	}
	nodeStore := BuildFakeNodeStore(nodes)

	tests := []struct {
		name         string
		resourceUID  string
		expectedDest string
		expectedKind string
	}{
		{
			name:         "ClusterPolicy",
			resourceUID:  "local-cluster/18b016fe-1931-4e80-95d1-d51d3b936e24",
			expectedDest: "local-cluster/8fc338fa-0591-44da-8a06-98ea7d74a7f7",
			expectedKind: "ClusterPolicy",
		},
		{
			name:         "namespaced Policy",
			resourceUID:  "local-cluster/777016fe-1931-4e80-95d1-d51d3b936e24",
			expectedDest: "local-cluster/777738fa-0591-44da-8a06-98ea7d74a7f7",
			expectedKind: "Policy",
		},
		{
			name:         "GeneratingPolicy",
			resourceUID:  "local-cluster/888016fe-1931-4e80-95d1-d51d3b936e24",
			expectedDest: "local-cluster/9fc338fa-0591-44da-8a06-98ea7d74a7f8",
			expectedKind: "GeneratingPolicy",
		},
		{
			name:         "NamespacedGeneratingPolicy",
			resourceUID:  "local-cluster/999016fe-1931-4e80-95d1-d51d3b936e24",
			expectedDest: "local-cluster/afc338fa-0591-44da-8a06-98ea7d74a7f9",
			expectedKind: "NamespacedGeneratingPolicy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("Test Kyverno " + tt.name)
			edges := CommonEdges(tt.resourceUID, nodeStore)
			if len(edges) != 1 {
				t.Fatalf("Expected 1 edge but got %d", len(edges))
			}

			expectedEdge := Edge{
				EdgeType:   "generatedBy",
				SourceUID:  tt.resourceUID,
				DestUID:    tt.expectedDest,
				SourceKind: "ConfigMap",
				DestKind:   tt.expectedKind,
			}

			AssertDeepEqual("edge", edges[0], expectedEdge, t)
		})
	}
}
