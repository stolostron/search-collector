// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

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
