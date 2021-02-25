/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	app "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
)

func TestTransformPlacementRule(t *testing.T) {
	var p app.PlacementRule
	UnmarshalFile("placementrule.json", &p, t)
	node := PlacementRuleResourceBuilder(&p).BuildNode()

	// Test only the fields that exist in placementrule - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementRule", t)
	AssertEqual("apigroup", node.Properties["apigroup"], "apps.open-cluster-management.io", t)
}

func TestTransformPlacementRuleWithClusterReplicas(t *testing.T) {
	var p app.PlacementRule
	UnmarshalFile("placementrule2.json", &p, t)
	node := PlacementRuleResourceBuilder(&p).BuildNode()

	// Test only the fields that exist in placementrule - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementRule", t)
	AssertEqual("apigroup", node.Properties["apigroup"], "apps.open-cluster-management.io", t)
	AssertEqual("replicas", node.Properties["replicas"], int32(5), t)
}
