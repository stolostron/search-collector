/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformStatefulSet(t *testing.T) {
	var s v1.StatefulSet
	UnmarshalFile("../../test-data/statefulset.json", &s, t)
	node := StatefulSetResource{&s}.BuildNode()

	// Test only the fields that exist in stateful set - the common test will test the other bits
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
}
