package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var labels = map[string]string{"app": "test", "fake": "true", "component": "testapp"}
var timestamp = machineryV1.Now()

// Helper function for creating a k8s resource to pass in to tests.
// In this case it's a pod.
func CreateGenericResource() machineryV1.Object {
	// Construct an object to test with, in this case a Pod with some of its fields blank.
	p := v1.Pod{}
	p.APIVersion = "v1"
	p.Name = "testpod"
	p.Namespace = "default"
	p.SelfLink = "/api/v1/namespaces/default/pods/testpod"
	p.UID = "00aa0000-00aa-00a0-a000-00000a00a0a0"
	p.ResourceVersion = "1000"
	p.CreationTimestamp = timestamp
	p.Labels = labels
	p.ClusterName = "TestCluster"
	return &p
}

func TestCommonProperties(t *testing.T) {

	res := CreateGenericResource()
	timeString := timestamp.String()

	cp := CommonProperties(res)

	// Test all the fields.
	AssertEqual("resourceVersion", cp["resourceVersion"], interface{}("1000"), t)
	AssertEqual("cluster", cp["cluster"], interface{}("TestCluster"), t)
	AssertEqual("name", cp["name"], interface{}("testpod"), t)
	AssertEqual("namespace", cp["namespace"], interface{}("default"), t)
	AssertEqual("selfLink", cp["selfLink"], interface{}("/api/v1/namespaces/default/pods/testpod"), t)
	AssertEqual("created", cp["created"], interface{}(timeString), t)

	noLabels := true
	for key, value := range cp["labels"].(map[string]string) {
		noLabels = false
		if labels[key] != value {
			t.Error("Incorrect label: " + key)
			t.Fail()
		}
	}

	if noLabels {
		t.Error("No labels found on resource")
		t.Fail()
	}
}
