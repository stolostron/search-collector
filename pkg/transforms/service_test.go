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

func TestTransformService(t *testing.T) {
	var s v1.Service
	UnmarshalFile("service.json", &s, t)
	node := ServiceResourceBuilder(&s).BuildNode()

	AssertEqual("kind", node.Properties["kind"], "Service", t)
}
