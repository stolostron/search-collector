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
	"time"

	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
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
	appNode.ResourceString = "applications"
	unstructuredInput := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "foobar",
			"metadata": map[string]interface{}{
				"uid": "1234",
			},
		},
	}
	unstructuredNode := GenericResourceBuilder(&unstructuredInput).BuildNode()
	unstructuredNode.ResourceString = "unstructured"

	var addonTyped agentv1.KlusterletAddonConfig
	var addonInput unstructured.Unstructured
	UnmarshalFile("klusterletaddonconfig.json", &addonInput, t)
	UnmarshalFile("klusterletaddonconfig.json", &addonTyped, t)

	addonNode := KlusterletAddonConfigResourceBuilder(&addonTyped).BuildNode()
	addonNode.ResourceString = "klusterletaddonconfigs"

	unstructGatekeeperConstraint := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "constraints.gatekeeper.sh/v1beta1",
			"kind":       "K8sRequiredLabels",
			"metadata": map[string]interface{}{
				"name": "ns-must-have-gk",
				"uid":  "783472a78",
			},
			"spec": map[string]interface{}{
				"enforcementAction": "dryrun",
				"other":             "value",
			},
			"status": map[string]interface{}{
				"auditTimestamp":  "2024-08-26T19:12:02Z",
				"totalViolations": 84,
			},
		},
	}
	gatekeeperPrinterColumns := []ExtractProperty{
		{Name: "enforcementAction", JSONPath: "{.spec.enforcementAction}"},
		{Name: "totalViolations", JSONPath: "{.status.totalViolations}"},
	}
	gatekeeperConstraintNode := GenericResourceBuilder(
		&unstructGatekeeperConstraint, gatekeeperPrinterColumns...,
	).BuildNode()
	gatekeeperConstraintNode.ResourceString = "k8srequiredlabels"
	gatekeeperConstraintNode.Properties["_isExternal"] = false
	gatekeeperConstraintNode.Properties["enforcementAction"] = "dryrun"
	gatekeeperConstraintNode.Properties["totalViolations"] = 84

	tests := []struct {
		name     string
		in       *Event
		expected NodeEvent
	}{
		{
			"Application create",
			&Event{
				Time:           ts,
				Operation:      Create,
				Resource:       &appInput,
				ResourceString: "applications",
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
				Time:           ts,
				Operation:      Delete,
				Resource:       &appInput,
				ResourceString: "applications",
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
				Time:           ts,
				Operation:      Create,
				Resource:       &unstructuredInput,
				ResourceString: "unstructured",
			},
			NodeEvent{
				Node:      unstructuredNode,
				Time:      ts,
				Operation: Create,
			},
		},
		{
			"KlusterletAddonConfig type create",
			&Event{
				Time:           ts,
				Operation:      Create,
				Resource:       &addonInput,
				ResourceString: "klusterletaddonconfigs",
			},
			NodeEvent{
				Node:      addonNode,
				Time:      ts,
				Operation: Create,
			},
		},
		{
			"Gatekeeper constraint create",
			&Event{
				Time:                     ts,
				Operation:                Create,
				Resource:                 &unstructGatekeeperConstraint,
				ResourceString:           "k8srequiredlabels",
				AdditionalPrinterColumns: gatekeeperPrinterColumns,
			},
			NodeEvent{
				Node:      gatekeeperConstraintNode,
				Time:      ts,
				Operation: Create,
			},
		},
	}

	go TransformRoutine(input, output)

	for _, test := range tests {
		input <- test.in
		actual := <-output
		test.expected.Node.Properties["kind_plural"] = test.in.ResourceString
		AssertDeepEqual(test.name, actual.Node, test.expected.Node, t)
		AssertEqual(test.name, actual.Time, test.expected.Time, t)
		AssertEqual(test.name, actual.Operation, test.expected.Operation, t)
	}
}
