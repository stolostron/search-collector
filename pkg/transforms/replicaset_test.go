/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2021 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformReplicaSet(t *testing.T) {
	var r v1.ReplicaSet
	UnmarshalFile("replicaset.json", &r, t)
	node := ReplicaSetResourceBuilder(&r).BuildNode()

	// Test only the fields that exist in replica set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
}
