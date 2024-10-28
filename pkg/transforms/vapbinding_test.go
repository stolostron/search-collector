// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
func TestVapBinding(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("vapbinding.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	node := VapBindingResourceBuilder(unstructured).BuildNode()

	AssertDeepEqual("validationActions", node.Properties["validationActions"], []string{"Deny", "Warn", "Audit"}, t)
	AssertEqual("_ownedByGatekeeper", node.Properties["_ownedByGatekeeper"], true, t)
	AssertEqual("policyName", node.Properties["policyName"], "demo-policy.example.com", t)
}

func TestVapBindingEdges(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("vapbinding.json", &object, t)

	vap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1",
			"kind":       "ValidatingAdmissionPolicy",
			"metadata": map[string]interface{}{
				"name": "demo-policy.example.com",
				"uid":  "7ed361c4-fa53-400c-8ae6-7ce6692012c3",
			},
		},
	}
	nodes := []Node{GenericResourceBuilder(vap).BuildNode()}
	nodeStore := BuildFakeNodeStore(nodes)

	vapb := &unstructured.Unstructured{Object: object}

	edges := VapBindingResourceBuilder(vapb).BuildEdges(nodeStore)
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge but got %d", len(edges))
	}

	edge := edges[0]
	expectedEdge := Edge{
		EdgeType:   "attachedTo",
		SourceUID:  "local-cluster/7385bbe4-031d-4cbe-833a-afd784526e6a",
		DestUID:    "local-cluster/7ed361c4-fa53-400c-8ae6-7ce6692012c3",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "ValidatingAdmissionPolicy",
	}
	AssertDeepEqual("edge", edge, expectedEdge, t)
}
