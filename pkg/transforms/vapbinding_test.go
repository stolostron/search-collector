// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"sort"
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
	AssertEqual("_ownedBy", node.Properties["_ownedBy"], "Gatekeeper", t)
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

func TestVapBindingEdgesGatekeeper(t *testing.T) {
	vapb := &unstructured.Unstructured{Object: map[string]interface{}{}}
	UnmarshalFile("vapb-gatekeeper.json", &vapb.Object, t)

	vap := &unstructured.Unstructured{Object: map[string]interface{}{}}
	UnmarshalFile("vap-gatekeeper.json", &vap.Object, t)

	constraint := &unstructured.Unstructured{Object: map[string]interface{}{}}
	UnmarshalFile("vap-gatekeeper-constraint.json", &constraint.Object, t)

	nodes := []Node{
		VapBindingResourceBuilder(vapb).BuildNode(),
		GenericResourceBuilder(vap).BuildNode(),
		GenericResourceBuilder(constraint).BuildNode(),
	}
	nodeStore := BuildFakeNodeStore(nodes)

	edges := VapBindingResourceBuilder(vapb).BuildEdges(nodeStore)
	if len(edges) != 3 {
		t.Fatalf("Expected 3 edges but got %d", len(edges))
	}

	edge := edges[0]
	expectedEdge := Edge{
		EdgeType:   "attachedTo",
		SourceUID:  "local-cluster/13e52147-2d8a-44ec-ab1a-11be247d4816",
		DestUID:    "local-cluster/3d21ea2d-3582-4aa8-85ff-f8dd83382b4d",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "ValidatingAdmissionPolicy",
	}
	AssertDeepEqual("edge", edge, expectedEdge, t)

	edge2 := edges[1]
	expectedEdge2 := Edge{
		EdgeType:   "uses",
		SourceUID:  "local-cluster/903d9fea-540a-4805-9ed2-f4e15e57f0ea",
		DestUID:    "local-cluster/3d21ea2d-3582-4aa8-85ff-f8dd83382b4d",
		SourceKind: "K8sRequiredLabels",
		DestKind:   "ValidatingAdmissionPolicy",
	}
	AssertDeepEqual("edge", edge2, expectedEdge2, t)

	edge3 := edges[2]
	expectedEdge3 := Edge{
		EdgeType:   "paramReferences",
		SourceUID:  "local-cluster/13e52147-2d8a-44ec-ab1a-11be247d4816",
		DestUID:    "local-cluster/903d9fea-540a-4805-9ed2-f4e15e57f0ea",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "K8sRequiredLabels",
	}
	AssertDeepEqual("edge", edge3, expectedEdge3, t)
}

func TestVapBindingEdgesSelector(t *testing.T) {
	vapb := &unstructured.Unstructured{Object: map[string]interface{}{}}
	UnmarshalFile("vapb-selector.json", &vapb.Object, t)

	vap := &unstructured.Unstructured{Object: map[string]interface{}{}}
	UnmarshalFile("vap-selector.json", &vap.Object, t)

	configMap1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "configmap1",
				"namespace": "policies-configs",
				"uid":       "a370cb76-a97c-4d9f-9249-c9bd8c2e4415",
				"labels": map[string]interface{}{
					"vap-config": "max-replicas",
					"random":     "configmap1",
				},
			},
			"data": map[string]interface{}{
				"maxReplicas.maxReplicas": "5",
			},
		},
	}

	configMap2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "configmap2",
				"namespace": "policies-configs",
				"uid":       "9768ff35-6d4c-4329-b3d6-b2929ab6d277",
				"labels": map[string]interface{}{
					"vap-config": "max-replicas",
					"random":     "configmap2",
				},
			},
			"data": map[string]interface{}{
				"maxReplicas.maxReplicas": "3",
			},
		},
	}

	configMapNoLabel := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "configmap-no-label",
				"namespace": "policies-configs",
				"uid":       "9ce38d78-3398-4226-9134-b31eb0bc30bc",
				"labels": map[string]interface{}{
					"random": "configmap-no-label",
				},
			},
			"data": map[string]interface{}{
				"maxReplicas.maxReplicas": "2",
			},
		},
	}

	configMapWrongNamespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "configmap-wrong-namespace",
				"namespace": "random-namespace",
				"uid":       "efdc566a-a1f8-4c9c-ab8a-4f8c9fc4b085",
				"labels": map[string]interface{}{
					"vap-config": "max-replicas",
					"random":     "config-wrong-namespace",
				},
			},
			"data": map[string]interface{}{
				"maxReplicas.maxReplicas": "16",
			},
		},
	}

	nodes := []Node{
		VapBindingResourceBuilder(vapb).BuildNode(),
		GenericResourceBuilder(vap).BuildNode(),
		GenericResourceBuilder(configMap1).BuildNode(),
		GenericResourceBuilder(configMap2).BuildNode(),
		GenericResourceBuilder(configMapNoLabel).BuildNode(),
		GenericResourceBuilder(configMapWrongNamespace).BuildNode(),
	}
	nodeStore := BuildFakeNodeStore(nodes)

	edges := VapBindingResourceBuilder(vapb).BuildEdges(nodeStore)
	if len(edges) != 3 {
		t.Fatalf("Expected 3 edges but got %d", len(edges))
	}

	// Sort the edges so the order is consistent for the assertions below
	sort.Slice(edges, func(i, j int) bool { return edges[i].DestUID < edges[j].DestUID })

	edge := edges[0]
	expectedEdge := Edge{
		EdgeType:   "attachedTo",
		SourceUID:  "local-cluster/8b0cfdca-27c0-48cb-a5be-bd376786498a",
		DestUID:    "local-cluster/2a13d661-9ea4-4ddd-9ca6-2a0fea714072",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "ValidatingAdmissionPolicy",
	}
	AssertDeepEqual("edge", edge, expectedEdge, t)

	edge2 := edges[1]
	expectedEdge2 := Edge{
		EdgeType:   "paramReferences",
		SourceUID:  "local-cluster/8b0cfdca-27c0-48cb-a5be-bd376786498a",
		DestUID:    "local-cluster/9768ff35-6d4c-4329-b3d6-b2929ab6d277",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "ConfigMap",
	}
	AssertDeepEqual("edge", edge2, expectedEdge2, t)

	edge3 := edges[2]
	expectedEdge3 := Edge{
		EdgeType:   "paramReferences",
		SourceUID:  "local-cluster/8b0cfdca-27c0-48cb-a5be-bd376786498a",
		DestUID:    "local-cluster/a370cb76-a97c-4d9f-9249-c9bd8c2e4415",
		SourceKind: "ValidatingAdmissionPolicyBinding",
		DestKind:   "ConfigMap",
	}
	AssertDeepEqual("edge", edge3, expectedEdge3, t)
}
