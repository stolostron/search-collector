/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	v1 "github.com/IBM/multicloud-operators-subscription/pkg/apis/app/v1alpha1"
)

func TestTransformSubscription(t *testing.T) {
	var s v1.Subscription
	UnmarshalFile("../../test-data/subscription.json", &s, t)
	node := SubscriptionResource{&s}.BuildNode()

	// Test only the fields that exist in stateful set - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Subscription", t)
	AssertEqual("packageFilterVersion", node.Properties["packageFilterVersion"], "1.x", t)
	AssertEqual("package", node.Properties["package"], "test-package", t)
	AssertEqual("channel", node.Properties["channel"], "testNs/test-channel", t)
}
