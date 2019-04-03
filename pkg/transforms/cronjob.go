/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

// Takes a *v1.CronJob and yields a Node
func transformCronJob(resource *v1.CronJob) Node {

	cronJob := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	cronJob.Properties["kind"] = "CronJob"
	cronJob.Properties["apigroup"] = "batch"
	cronJob.Properties["active"] = int64(len(resource.Status.Active))
	cronJob.Properties["schedule"] = resource.Spec.Schedule
	cronJob.Properties["lastSchedule"] = ""
	if resource.Status.LastScheduleTime != nil {
		cronJob.Properties["lastSchedule"] = resource.Status.LastScheduleTime.UTC().Format(time.RFC3339)
	}
	cronJob.Properties["suspend"] = false
	if resource.Spec.Suspend != nil {
		cronJob.Properties["suspend"] = *resource.Spec.Suspend
	}

	return cronJob
}
