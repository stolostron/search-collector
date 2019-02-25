package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Pod and extracts the subset of properties that we care about, yielding a transforms.PodNode
func TransformPod(resource *v1.Pod) rg.Node {

	// Count the number of restarts. We define the number of Pod restarts to be the sum of the container restarts of containers in the Pod.
	var restarts uint = 0
	for _, containerStatus := range resource.Status.ContainerStatuses {
		restarts += uint(containerStatus.RestartCount)
	}

	props := CommonProperties(resource) // Start off with the common properties

	// Extract the properties specific to this type
	props["hostIP"] = resource.Status.HostIP
	props["podIP"] = resource.Status.PodIP
	props["restarts"] = restarts
	props["startedAt"] = resource.Status.StartTime.String()
	props["status"] = string(resource.Status.Phase)

	// Form these properties into an rg.Node
	return rg.Node{
		Label:      "Pod",
		Properties: props,
	}
}
