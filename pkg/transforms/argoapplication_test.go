/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	app "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

func TestTransformArgoApplication(t *testing.T) {
	var a app.Application
	UnmarshalFile("argoapplication.json", &a, t)
	node := ArgoApplicationResourceBuilder(&a).BuildNode()

	// Test only the fields that exist in application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
}
