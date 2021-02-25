/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	app "github.com/open-cluster-management/multicloud-operators-subscription-release/pkg/apis/apps/v1"
)

//TODO: Might have to update the json for testing once we have a live example
func TestTransformAppHelmCR(t *testing.T) {
	var a app.HelmRelease

	UnmarshalFile("apphelmcr.json", &a, t)

	node := AppHelmCRResourceBuilder(&a).BuildNode()

	// Test only the fields that exist in HelmRelease - the common test will test the other bits
	AssertEqual("name", node.Properties["name"], "testAppHelmCR", t)
	AssertEqual("kind", node.Properties["kind"], "HelmRelease", t)
}
