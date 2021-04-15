// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"
)

func TestTransformArgoApplication(t *testing.T) {
	var a ArgoApplication
	UnmarshalFile("argoapplication.json", &a, t)
	node := ArgoApplicationResourceBuilder(&a).BuildNode()

	// Test only the fields that exist in application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
	AssertEqual("destinationName", node.Properties["destinationName"], "local-cluster", t)
	AssertEqual("destinationNamespace", node.Properties["destinationNamespace"], "argo-helloworld", t)
	AssertEqual("destinationServer", node.Properties["destinationServer"], "https://kubernetes.default.svc", t)
	AssertEqual("path", node.Properties["path"], "helloworld", t)
	AssertEqual("chart", node.Properties["chart"], "hello-chart", t)
	AssertEqual("repoURL", node.Properties["repoURL"], "https://github.com/fxiang1/app-samples", t)
	AssertEqual("targetRevision", node.Properties["targetRevision"], "HEAD", t)
	AssertEqual("status", node.Properties["status"], "Healthy", t)
}
