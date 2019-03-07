package transforms

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Pod and extracts the subset of properties that we care about, yielding a transforms.PodNode
func TransformPod(resource *v1.Pod) Node {

	// Count the number of restarts. We define the number of Pod restarts to be the sum of the container restarts of containers in the Pod.
	var restarts uint
	for _, containerStatus := range resource.Status.ContainerStatuses {
		restarts += uint(containerStatus.RestartCount)
	}

	pod := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	pod.Properties["kind"] = "Pod"
	pod.Properties["hostIP"] = resource.Status.HostIP
	pod.Properties["podIP"] = resource.Status.PodIP
	pod.Properties["restarts"] = restarts
	pod.Properties["status"] = string(resource.Status.Phase)
	pod.Properties["startedAt"] = ""
	if resource.Status.StartTime != nil {
		pod.Properties["startedAt"] = resource.Status.StartTime.UTC().Format(time.RFC3339)
	}

	return pod
}
