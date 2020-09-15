/*
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"testing"

	v1 "github.com/openshift/api/apps/v1"
)

func TestTransformDeploymentConfig(t *testing.T) {
	var d v1.DeploymentConfig
	UnmarshalFile("deployment.json", &d, t)
	node := DeploymentConfigResourceBuilder(&d).BuildNode()

	// Test only the fields that exist in deployment
	AssertEqual("available", node.Properties["available"], int64(1), t)
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
	AssertEqual("ready", node.Properties["ready"], int64(1), t)
}
