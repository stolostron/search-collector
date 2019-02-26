package transforms

import (
	rg "github.com/redislabs/redisgraph-go"
	v1 "k8s.io/api/batch/v1beta1"
)

// Takes a *v1.CronJob and yields a rg.Node
func TransformCronJob(resource *v1.CronJob) rg.Node {

	props := CommonProperties(resource) // Start off with the common properties

	// Extract the properties specific to this type
	props["active"] = len(resource.Status.Active)
	props["lastSchedule"] = resource.Status.LastScheduleTime.String()
	props["schedule"] = resource.Spec.Schedule
	props["suspend"] = resource.Spec.Suspend

	// Form these properties into an rg.Node
	return rg.Node{
		Label:      "CronJob",
		Properties: props,
	}
}
