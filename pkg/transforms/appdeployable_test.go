/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"testing"

	app "github.com/stolostron/multicloud-operators-deployable/pkg/apis/apps/v1"
)

func TestTransformAppDeployable(t *testing.T) {
	var d app.Deployable
	UnmarshalFile("appdeployable.json", &d, t)
	node := AppDeployableResourceBuilder(&d).BuildNode()

	// Test only the fields that exist in deployable - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Deployable", t)
	AssertEqual("apigroup", node.Properties["apigroup"], "apps.open-cluster-management.io", t)
}
