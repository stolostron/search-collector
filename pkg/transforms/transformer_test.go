/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"
	"time"

	v1 "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformRoutine(t *testing.T) {
	input := make(chan *Event)
	output := make(chan NodeEvent)

	// generate input and output test nodes
	ts := time.Now().Unix()
	var appTyped v1.Application
	var appInput unstructured.Unstructured
	UnmarshalFile("../../test-data/application.json", &appTyped, t)
	UnmarshalFile("../../test-data/application.json", &appInput, t)
	appNode := ApplicationResource{&appTyped}.BuildNode()

	unstructuredInput := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "foobar",
			"metadata": map[string]interface{}{
				"uid": "1234",
			},
		},
	}
	unstructuredNode := UnstructuredResource{&unstructuredInput}.BuildNode()

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

	go transformRoutine(input, output, nil)

	for _, test := range tests {
		input <- test.in
		actual := <-output

		AssertDeepEqual(test.name, actual.Node, test.expected.Node, t)
		AssertEqual(test.name, actual.Time, test.expected.Time, t)
		AssertEqual(test.name, actual.Operation, test.expected.Operation, t)
	}
}
