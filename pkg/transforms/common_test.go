/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
*/

package transforms

import (
	"testing"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var labels = map[string]string{"app": "test", "fake": "true", "component": "testapp"}
var timestamp = machineryV1.Now()

// Helper function for creating a k8s resource to pass in to tests.
// In this case it's a pod.
func CreateGenericResource() machineryV1.Object {
	// Construct an object to test with, in this case a Pod with some of its fields blank.
	p := v1.Pod{}
	p.APIVersion = "v1"
	p.Name = "testpod"
	p.Namespace = "default"
	p.UID = "00aa0000-00aa-00a0-a000-00000a00a0a0"
	p.CreationTimestamp = timestamp
	p.Labels = labels
	p.Annotations = map[string]string{"hello": "world"}
	return &p
}

func TestCommonProperties(t *testing.T) {
	// Establish the config
	config.InitConfig()
	config.Cfg.CollectAnnotations = true

	defer func() {
		config.Cfg.CollectAnnotations = false
	}()

	res := CreateGenericResource()
	timeString := timestamp.UTC().Format(time.RFC3339)

	cp := commonProperties(res)

	// Test all the fields.
	AssertEqual("name", cp["name"], interface{}("testpod"), t)
	AssertEqual("namespace", cp["namespace"], interface{}("default"), t)
	AssertEqual("created", cp["created"], interface{}(timeString), t)
	AssertEqual("annotation", cp["annotation"].(map[string]string)["hello"], "world", t)

	noLabels := true
	for key, value := range cp["label"].(map[string]string) {
		noLabels = false
		if labels[key] != value {
			t.Error("Incorrect label: " + key)
			t.Fail()
		}
	}

	if noLabels {
		t.Error("No labels found on resource")
		t.Fail()
	}
}
