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

	v1 "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
)

func TestTransformSubscription(t *testing.T) {
	var s v1.Subscription
	UnmarshalFile("subscription.json", &s, t)
	node := SubscriptionResource{&s}.BuildNode()

	// Test only the fields that exist in subscription - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Subscription", t)
	AssertEqual("packageFilterVersion", node.Properties["packageFilterVersion"], "1.x", t)
	AssertEqual("package", node.Properties["package"], "test-package", t)
	AssertEqual("channel", node.Properties["channel"], "testNs/test-channel", t)
}

func TestTransformSubscriptionWithTimeWindow(t *testing.T) {
	var s v1.Subscription
	UnmarshalFile("subscription2.json", &s, t)
	node := SubscriptionResource{&s}.BuildNode()

	// Test optional fields that exist in subscription - the common test will test the other bits
	AssertEqual("timeWindow", node.Properties["timeWindow"], "active", t)
	AssertEqual("_gitbranch", node.Properties["_gitbranch"], "master", t)
	AssertEqual("_gitpath", node.Properties["_gitpath"], "helloworld", t)
	AssertEqual("_gitcommit", node.Properties["_gitcommit"], "d67d8e10dcfa41dddcac14952e9872e1dfece06f", t)
}

func TestTransformSubscriptionWithLocalPlacement(t *testing.T) {
	var s v1.Subscription
	UnmarshalFile("subscription3.json", &s, t)
	node := SubscriptionResource{&s}.BuildNode()

	// Test optional fields that exist in subscription - the common test will test the other bits
	AssertEqual("localPlacement", node.Properties["localPlacement"], "True", t)
}
