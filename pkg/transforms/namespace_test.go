/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformNamespace(t *testing.T) {
	var n v1.Namespace
	UnmarshalFile("namespace.json", &n, t)
	node := NamespaceResourceBuilder(&n).BuildNode()

	// Test only the fields that exist in namespace - the common test will test the other bits
	AssertEqual("status", node.Properties["status"], "Active", t)
}
