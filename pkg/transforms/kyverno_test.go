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
