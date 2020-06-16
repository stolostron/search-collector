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

	app "sigs.k8s.io/application/api/v1beta1"
)

func TestTransformApplication(t *testing.T) {
	var a app.Application
	UnmarshalFile("application.json", &a, t)
	node := ApplicationResource{&a}.BuildNode()

	// Test only the fields that exist in application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
	AssertEqual("dashboard", node.Properties["dashboard"], "https://0.0.0.0:8443/grafana/dashboard/test", t)
}
