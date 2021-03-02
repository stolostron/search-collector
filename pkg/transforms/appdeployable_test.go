/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	app "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

func TestTransformAppDeployable(t *testing.T) {
	var d app.Deployable
	UnmarshalFile("appdeployable.json", &d, t)
	node := AppDeployableResourceBuilder(&d).BuildNode()

	// Test only the fields that exist in deployable - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Deployable", t)
	AssertEqual("apigroup", node.Properties["apigroup"], "apps.open-cluster-management.io", t)
}
