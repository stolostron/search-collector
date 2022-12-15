// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"
)

func TestTransformArgoApplication(t *testing.T) {
	var a ArgoApplication
	UnmarshalFile("argoapplication.json", &a, t)
	argoApplicationResource := ArgoApplicationResourceBuilder(&a)

	node := argoApplicationResource.BuildNode()

	// Test only the fields that exist in application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
	AssertEqual("applicationSet", node.Properties["applicationSet"], "helloworld-set", t)
	AssertEqual("hostingResource", node.Properties["_hostingResource"], "ApplicationSet/openshift-gitops/bgdk-app-set", t)
	AssertEqual("destinationName", node.Properties["destinationName"], "local-cluster", t)
	AssertEqual("destinationNamespace", node.Properties["destinationNamespace"], "argo-helloworld", t)
	AssertEqual("destinationServer", node.Properties["destinationServer"], "https://kubernetes.default.svc", t)
	AssertEqual("path", node.Properties["path"], "helloworld", t)
	AssertEqual("chart", node.Properties["chart"], "hello-chart", t)
	AssertEqual("repoURL", node.Properties["repoURL"], "https://github.com/fxiang1/app-samples", t)
	AssertEqual("targetRevision", node.Properties["targetRevision"], "HEAD", t)
	AssertEqual("healthStatus", node.Properties["healthStatus"], "Missing", t)
	AssertEqual("syncStatus", node.Properties["syncStatus"], "OutOfSync", t)

	// Test argocd application status conditions count
	// message in SyncError is truncated to 512 + 3("...")
	// message in InvalidSpecError is not truncated.
	AssertEqual("ConditionSyncError", len(node.Properties["_conditionSyncError"].(string)), 515, t)

	AssertEqual("ConditionInvalidSpecError", len(node.Properties["_conditionInvalidSpecError"].(string)), 38, t)

	AssertEqual("ConditionOperationError", len(node.Properties["_conditionOperationError"].(string)), 515, t)

	// Build a fake NodeStore with nodes needed to generate edges.
	nodes := []Node{{
		UID:        "uuid-123-namespace",
		Properties: map[string]interface{}{"kind": "Namespace", "namespace": "_NONE", "name": "bgd"},
	}, {
		UID:        "uuid-123-service",
		Properties: map[string]interface{}{"kind": "Service", "namespace": "bgd", "name": "bgd"},
	}, {
		UID:        "uuid-123-deployment",
		Properties: map[string]interface{}{"kind": "Deployment", "namespace": "bgd", "name": "bgd"},
	}, {
		UID:        "uuid-123-route",
		Properties: map[string]interface{}{"kind": "Route", "namespace": "bgd", "name": "bgd"},
	}}

	nodeStore := BuildFakeNodeStore(nodes)

	edges := argoApplicationResource.BuildEdges(nodeStore)

	AssertEqual("edges", len(edges), 4, t)
}
