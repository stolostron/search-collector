/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"testing"

	app "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
)

func TestTransformChannel(t *testing.T) {
	var c app.Channel
	UnmarshalFile("channel.json", &c, t)
	node := ChannelResourceBuilder(&c).BuildNode()

	// Test only the fields that exist in channel - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Channel", t)
}
