/*
 * (C) Copyright IBM Corporation 2019 All Rights Reserved
 * Copyright (c) 2020 Red Hat, Inc.
 * Copyright Contributors to the Open Cluster Management project
 */

package transforms

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	app "sigs.k8s.io/application/api/v1beta1"
)

func TestTransformRoutine(t *testing.T) {
	input := make(chan *Event)
	output := make(chan NodeEvent)

	// generate input and output test nodes
	ts := time.Now().Unix()
	var appTyped app.Application
	var appInput unstructured.Unstructured
	UnmarshalFile("application.json", &appTyped, t)
	UnmarshalFile("application.json", &appInput, t)
	appNode := ApplicationResourceBuilder(&appTyped).BuildNode()

	unstructuredInput := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "foobar",
			"metadata": map[string]interface{}{
				"uid": "1234",
			},
		},
	}
	unstructuredNode := GenericResourceBuilder(&unstructuredInput).BuildNode()

	var tests = []struct {
		name     string
		in       *Event
		expected NodeEvent
	}{
		{
			"Application create",
			&Event{
				Time:      ts,
				Operation: Create,
				Resource:  &appInput,
			},
			NodeEvent{
				Node:      appNode,
				Time:      ts,
				Operation: Create,
			},
		},
		{
			"Application delete",
			&Event{
				Time:      ts,
				Operation: Delete,
				Resource:  &appInput,
			},
			NodeEvent{
				Node:      appNode,
				Time:      ts,
				Operation: Delete,
			},
		},
		{
			"Unknown type create",
			&Event{
				Time:      ts,
				Operation: Create,
				Resource:  &unstructuredInput,
			},
			NodeEvent{
				Node:      unstructuredNode,
				Time:      ts,
				Operation: Create,
			},
		},
	}

	go TransformRoutine(input, output)

	for _, test := range tests {
		input <- test.in
		actual := <-output

		AssertDeepEqual(test.name, actual.Node, test.expected.Node, t)
		AssertEqual(test.name, actual.Time, test.expected.Time, t)
		AssertEqual(test.name, actual.Operation, test.expected.Operation, t)
	}
}
