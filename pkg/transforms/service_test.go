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

func TestTransformService(t *testing.T) {
	var s v1.Service
	UnmarshalFile("../../test-data/service.json", &s, t)
	node := ServiceResource{&s}.BuildNode()

	AssertEqual("kind", node.Properties["kind"], "Service", t)
}
