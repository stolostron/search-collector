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
				"validationFailureAction": "Deny",
				"random":                  "value",
			},
		},
	}
	rv := KyvernoPolicyResourceBuilder(&p)
	node := rv.node

	AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], "Deny", t)
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
				"validationFailureAction": "Deny",
				"random":                  "value",
				"background":              false,
				"admission":               false,
			},
		},
	}
	rv := KyvernoPolicyResourceBuilder(&p)
	node := rv.node

	AssertEqual("validationFailureAction", node.Properties["validationFailureAction"], "Deny", t)
	AssertEqual("background", node.Properties["background"], false, t)
	AssertEqual("admission", node.Properties["admission"], false, t)
	AssertEqual("severity", node.Properties["severity"], "critical", t)
}
