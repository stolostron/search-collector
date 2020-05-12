/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	mcm "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

func TestTransformDeployable(t *testing.T) {
	var d mcm.Deployable
	UnmarshalFile("deployable.json", &d, t)
	node := DeployableResource{&d}.BuildNode()

	// Test only the fields that exist in deployable - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Deployable", t)
}
