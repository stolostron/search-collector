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

func TestEvent(t *testing.T) {
	var e v1.Event
	UnmarshalFile("../../test-data/event.json", &e, t)
	node := EventResource{&e}.BuildNode()

	// Test only the fields that exist in VulnerabilityPolicy - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Event", t)
	AssertEqual("source", node.Properties["source"], "vulnerabilitypolicy-controller", t)
	AssertEqual("InvolvedObject.kind", node.Properties["InvolvedObject.kind"], "VulnerabilityPolicy", t)
	AssertEqual("InvolvedObject.name", node.Properties["InvolvedObject.name"], "va-policy-example2", t)
	AssertEqual("InvolvedObject.namespace", node.Properties["InvolvedObject.namespace"], "kube-system", t)
	AssertEqual("InvolvedObject.uid", node.Properties["InvolvedObject.uid"], "9ad2d2e0-f04b-11e9-ba0f-0016ac10172d", t)
	AssertEqual("message", node.Properties["message"], "Detected pod(uid:eb790c2e-361f-11e9-85ca-00163e019656) container default/ubi7-2-pod/ubi7-2-pod", t)
	AssertEqual("type", node.Properties["type"], "Warning", t)
	AssertEqual("message.uid", node.Properties["message.uid"], "eb790c2e-361f-11e9-85ca-00163e019656", t)
	AssertEqual("message.namespace", node.Properties["message.namespace"], "default", t)
	AssertEqual("message.resourceName", node.Properties["message.resourceName"], "ubi7-2-pod", t)
	AssertEqual("message.containerName", node.Properties["message.containerName"], "ubi7-2-pod", t)

}
