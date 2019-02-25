package transforms

import "k8s.io/api/core/v1"

// MCM Search representation of a pod to be put into graphDB
type PodNode struct {
	CommonNodeProperties
	HostIP    string `json: hostIP`
	PodIP     string `json: podIP`
	Restarts  uint   `json: restarts`
	StartedAt string `json: startedAt`
	Status    string `json: status`
}

// Takes a *v1.Pod and extracts the subset of properties that we care about, yielding a transforms.PodNode
func TransformPod(resource *v1.Pod) PodNode {

	// Count the number of restarts. We define the number of Pod restarts to be the sum of the container restarts of containers in the Pod.
	var restarts uint = 0
	for _, containerStatus := range resource.Status.ContainerStatuses {
		restarts += uint(containerStatus.RestartCount)
	}

	return PodNode{
		CommonNodeProperties: TransformCommon(resource),
		HostIP:               resource.Status.HostIP,
		PodIP:                resource.Status.PodIP,
		Restarts:             restarts,
		StartedAt:            resource.Status.StartTime.String(),
		Status:               string(resource.Status.Phase),
	}
}
