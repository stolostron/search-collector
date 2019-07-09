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

	v1 "k8s.io/api/core/v1"
)

func TestTransformNode(t *testing.T) {
	var n v1.Node
	UnmarshalFile("../../test-data/node.json", &n, t)
	node := NodeResource{&n}.BuildNode()

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("architecture", node.Properties["architecture"], "amd64", t)
	AssertEqual("cpu", node.Properties["cpu"], int64(8), t)
	AssertEqual("osImage", node.Properties["osImage"], "Ubuntu 16.04.5 LTS", t)
	AssertDeepEqual("role", node.Properties["role"], []string{"etcd", "management", "master", "proxy", "va"}, t)
}
