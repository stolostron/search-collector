package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var sTime = machineryV1.Now()

// Helper to create a resource for use in testing.
func CreatePod() *v1.Pod {
	// Construct an object to test with, in this case a Pod with some of its fields blank.
	p := v1.Pod{}
	p.Kind = "Pod"
	p.APIVersion = "v1"
	p.Name = "testpod"
	p.Namespace = "default"
	p.SelfLink = "/api/v1/namespaces/default/pods/testpod"
	p.UID = "00aa0000-00aa-00a0-a000-00000a00a0a0"
	p.ResourceVersion = "1000"
	p.CreationTimestamp = machineryV1.Now()
	p.Labels = map[string]string{"app": "test", "fake": "true", "component": "testapp"}
	p.ClusterName = "TestCluster"

	// Assemble PodStatus
	status := v1.PodStatus{}
	status.Phase = v1.PodRunning
	status.HostIP = "1.2.3.4"
	status.PodIP = "10.2.3.4"
	cStatus1 := v1.ContainerStatus{
		RestartCount: 0,
	}
	cStatus2 := v1.ContainerStatus{
		RestartCount: 1,
	}
	cStatus3 := v1.ContainerStatus{
		RestartCount: 2,
	}
	status.ContainerStatuses = []v1.ContainerStatus{cStatus1, cStatus2, cStatus3}
	status.StartTime = &sTime
	p.Status = status

	return &p
}

func TestTransformPod(t *testing.T) {

	res := CreatePod()

	tp := TransformPod(res)

	// Test only the fields that exist in pods - the common test will test the other bits
	AssertEqual("kind", tp.Properties["kind"], "Pod", t)
	AssertEqual("hostIP", tp.Properties["hostIP"], "1.2.3.4", t)
	AssertEqual("podIP", tp.Properties["podIP"], "10.2.3.4", t)
	AssertEqual("restarts", tp.Properties["restarts"], uint(3), t)
	AssertEqual("startedAt", tp.Properties["startedAt"], sTime.String(), t)
	AssertEqual("status", tp.Properties["status"], string(v1.PodRunning), t)

}
