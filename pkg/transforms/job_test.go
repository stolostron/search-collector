package transforms

import (
	"testing"

	v1 "k8s.io/api/batch/v1"
)

func TestTransformJob(t *testing.T) {
	var j v1.Job
	UnmarshalFile("../../test-data/job.json", &j, t)
	node := transformJob(&j)

	// Test only the fields that exist in job - the common test will test the other bits
	AssertEqual("successful", node.Properties["successful"], int32(1), t)
	AssertEqual("completions", node.Properties["completions"], int32(1), t)
	AssertEqual("parallelism", node.Properties["parallelism"], int32(1), t)
}
