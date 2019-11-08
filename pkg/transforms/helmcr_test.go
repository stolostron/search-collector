/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	app "github.ibm.com/IBMMulticloudPlatform/helm-crd/pkg/apis/helm.bitnami.com/v1"
)

func TestTransformHelmCR(t *testing.T) {
	var h app.HelmRelease

	UnmarshalFile("../../test-data/helmcr.json", &h, t)

	node := HelmCRResource{&h}.BuildNode()

	// Test only the fields that exist in HelmRelease - the common test will test the other bits
	AssertEqual("name", node.Properties["name"], "testHelmCR", t)
	AssertEqual("kind", node.Properties["kind"], "HelmRelease", t)
}
