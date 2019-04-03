package transforms

import (
	v1 "k8s.io/api/batch/v1"
)

// Takes a *v1.Job and yields a Node
func transformJob(resource *v1.Job) Node {

	job := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	job.Properties["kind"] = "Job"
	job.Properties["apigroup"] = "batch"
	job.Properties["successful"] = int64(resource.Status.Succeeded)
	job.Properties["completions"] = int64(0)
	if resource.Spec.Completions != nil {
		job.Properties["completions"] = int64(*resource.Spec.Completions)
	}
	job.Properties["parallelism"] = int64(0)
	if resource.Spec.Completions != nil {
		job.Properties["parallelism"] = int64(*resource.Spec.Parallelism)
	}

	return job
}
