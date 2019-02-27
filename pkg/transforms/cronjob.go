package transforms

import (
	v1 "k8s.io/api/batch/v1beta1"
)

// Takes a *v1.CronJob and yields a Node
func TransformCronJob(resource *v1.CronJob) Node {

	cronJob := CommonProperties(resource) // Start off with the common properties

	// Extract the properties specific to this type
	cronJob.properties["kind"] = "CronJob"
	cronJob.properties["active"] = len(resource.Status.Active)
	cronJob.properties["lastSchedule"] = resource.Status.LastScheduleTime.String()
	cronJob.properties["schedule"] = resource.Spec.Schedule
	cronJob.properties["suspend"] = resource.Spec.Suspend

	return cronJob
}
