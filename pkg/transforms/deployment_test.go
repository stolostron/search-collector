/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformDeployment(t *testing.T) {
	var d v1.Deployment
	UnmarshalFile("deployment.json", &d, t)
	node := DeploymentResource{&d}.BuildNode()

	// Test only the fields that exist in deployment - the common test will test the other bits
	AssertEqual("available", node.Properties["available"], int64(1), t)
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
	AssertEqual("ready", node.Properties["ready"], int64(1), t)
}
