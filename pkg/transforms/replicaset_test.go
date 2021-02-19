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

func TestTransformReplicaSet(t *testing.T) {
	var r v1.ReplicaSet
	UnmarshalFile("replicaset.json", &r, t)
	node := ReplicaSetResourceBuilder(&r).BuildNode()

	// Test only the fields that exist in replica set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
}

func TestReplicaSetBuildEdges(t *testing.T) {
	var rs v1.ReplicaSet
	UnmarshalFile("replicaset.json", &rs, t)

	store := NodeStore{
		ByUID:               make(map[string]Node),
		ByKindNamespaceName: make(map[string]map[string]map[string]Node),
	}

	edges := ReplicaSetResourceBuilder(&rs).BuildEdges(store)

	AssertEqual("ReplicaSet has no edges:", len(edges), 0, t)
}
