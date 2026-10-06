// Copyright (c) 2025 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformGKConstraintNonCompliant(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("gk-constraint-livenessprobe.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	priority := 0
	constraintResource := GkConstraintResourceBuilder(unstructured,
		ExtractProperty{Name: "enforcementAction", JSONPath: "{.spec.enforcementAction}", Priority: &priority},
		ExtractProperty{Name: "totalViolations", JSONPath: "{.status.totalViolations}", Priority: &priority},
	)

	node := constraintResource.BuildNode()

	// Only test the fields specific to Gatekeeper Constraints
	AssertEqual("compliant", node.Properties["compliant"], "NonCompliant", t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], false, t)
	AssertEqual("enforcementAction", node.Properties["enforcementAction"], "dryrun", t)
	AssertEqual("totalViolations", node.Properties["totalViolations"], float64(3), t)
	obj1 := relatedObject{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "fake-deployment",
	}
	obj2 := relatedObject{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "multicluster-engine",
		Name:      "provider-credential-controller",
	}
	AssertEqual("relObjs", node.GetMetadata("relObjs"),
		"["+obj1.String()+" "+obj2.String()+"]", t)
}

func TestTransformGKConstraintCompliant(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("gk-constraint-requiredlabels.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	constraintResource := GkConstraintResourceBuilder(unstructured)

	node := constraintResource.BuildNode()

	// Only test the fields specific to Gatekeeper Constraints
	AssertEqual("compliant", node.Properties["compliant"], "Compliant", t)
	AssertEqual("_isExternal", node.Properties["_isExternal"], false, t)
	AssertEqual("relObjs", node.GetMetadata("relObjs"), "", t)
}

func TestGkConstraintBuildEdges_PrefersGroupAwareLookup(t *testing.T) {
	constraintNode := Node{
		UID: "cluster/constraint-uid",
		Properties: map[string]interface{}{
			"kind": "K8sRequiredLabels",
		},
		Metadata: map[string]any{
			"relObjs": []relatedObject{{
				Group:    "config.openshift.io",
				Version:  "v1",
				Kind:     "Network",
				Name:     "cluster",
				EdgeType: noncompliantEdge,
			}},
		},
	}

	configNetwork := Node{UID: "cluster/config-network-uid", Properties: map[string]interface{}{"kind": "Network", "name": "cluster", "apigroup": "config.openshift.io"}}
	operatorNetwork := Node{UID: "cluster/operator-network-uid", Properties: map[string]interface{}{"kind": "Network", "name": "cluster", "apigroup": "operator.openshift.io"}}

	ns := NodeStore{
		ByUID: map[string]Node{
			constraintNode.UID:  constraintNode,
			configNetwork.UID:   configNetwork,
			operatorNetwork.UID: operatorNetwork,
		},
		ByKindNamespaceName: map[string]map[string]map[string]Node{
			"Network": {
				"_NONE": {
					"cluster": operatorNetwork,
				},
			},
		},
		ByGroupKindNamespaceName: map[string]map[string]map[string]map[string]Node{
			"config.openshift.io": {
				"Network": {
					"_NONE": {
						"cluster": configNetwork,
					},
				},
			},
			"operator.openshift.io": {
				"Network": {
					"_NONE": {
						"cluster": operatorNetwork,
					},
				},
			},
		},
	}

	edges := GkConstraintResource{node: constraintNode}.BuildEdges(ns)
	assert.Len(t, edges, 1)
	AssertEqual("destUID", edges[0].DestUID, configNetwork.UID, t)
}
