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

func TestTransformNamespace(t *testing.T) {
	var n v1.Namespace
	UnmarshalFile("../../test-data/namespace.json", &n, t)
	node := transformNamespace(&n)

	// Test only the fields that exist in namespace - the common test will test the other bits
	AssertEqual("status", node.Properties["status"], "Active", t)
}
