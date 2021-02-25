/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	mcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
)

func TestTransformPlacementBinding(t *testing.T) {
	var p mcm.PlacementBinding
	UnmarshalFile("placementbinding.json", &p, t)
	node := PlacementBindingResourceBuilder(&p).BuildNode()

	// Test only the fields that exist in placementbinding - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "PlacementBinding", t)
	AssertEqual("placementpolicy", node.Properties["placementpolicy"], "foo-test (PlacementPolicy)", t)
	AssertDeepEqual("subject", node.Properties["subject"], []string{"foo-test (Deployable)"}, t)
}
