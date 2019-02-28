package transforms

import (
	v1 "k8s.io/api/batch/v1"
)

// Takes a *v1.Job and yields a Node
func TransformJob(resource *v1.Job) Node {

	job := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	job.Properties["kind"] = "Job"
	job.Properties["completions"] = resource.Spec.Completions
	job.Properties["parallelism"] = resource.Spec.Parallelism
	job.Properties["successful"] = resource.Status.Succeeded

	return job
}
