// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_genericResourceFromConfig(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("clusterserviceversion.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "advanced-cluster-management.v2.9.0", t)
	AssertEqual("kind", node.Properties["kind"], "ClusterServiceVersion", t)
	AssertEqual("namespace", node.Properties["namespace"], "open-cluster-management", t)
	AssertEqual("created", node.Properties["created"], "2023-08-23T15:54:22Z", t)

	// Verify properties defined in the transform config
	AssertEqual("display", node.Properties["display"], "Advanced Cluster Management for Kubernetes", t)
	AssertEqual("phase", node.Properties["phase"], "Succeeded", t)
	AssertEqual("version", node.Properties["version"], "2.9.0", t)

}

func Test_genericResourceFromConfigVM(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachine.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "rhel9-gitops", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachine", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-04-30T16:22:02Z", t)

	// Verify properties defined in the transform config
	AssertEqual("display", node.Properties["status"], "Running", t)
	AssertEqual("phase", node.Properties["ready"], "True", t)

}
