/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	app "github.com/open-cluster-management/channel/pkg/apis/app/v1alpha1"
)

func TestTransformChannel(t *testing.T) {
	var c app.Channel
	UnmarshalFile("../../test-data/channel.json", &c, t)
	node := ChannelResource{&c}.BuildNode()

	// Test only the fields that exist in channel - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Channel", t)
}
