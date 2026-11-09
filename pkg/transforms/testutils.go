/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/
// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

// Contains utils for use in testing.
package transforms

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	sanitize "github.com/kennygrant/sanitize"
)

// UnmarshalFile takes a file path and unmarshals it into the given resource type.
func UnmarshalFile(filepath string, resourceType interface{}, t *testing.T) {
	// open given filepath string
	rawBytes, err := os.ReadFile("../../test-data/" + sanitize.Name(filepath))
	if err != nil {
		t.Fatal("Unable to read test data", err)
	}

	// unmarshal file into given resource type
	err = json.Unmarshal(rawBytes, resourceType)
	if err != nil {
		t.Fatalf("Unable to unmarshal json to type %T %s", resourceType, err)
	}
}

// Checks whether two things are equal. If they are not, prints an error and fails the test.
// If they are equal, there is no effect.
// NOTE: You can only use this to compare types that are comparable under the hood.
func AssertEqual(property string, actual, expected interface{}, t *testing.T) {
	if expected != actual {
		t.Errorf("%s EXPECTED: %T %v\n", property, expected, expected)
		t.Errorf("%s ACTUAL: %T %v\n", property, actual, actual)
		t.Fail()
	}
}

func AssertDeepEqual(property string, actual, expected interface{}, t *testing.T) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s EXPECTED: %T %v\n", property, expected, expected)
		t.Errorf("%s ACTUAL: %T %v\n", property, actual, actual)
		t.Fail()
	}
}

func BuildFakeNodeStore(nodes []Node) NodeStore {
	byUID := make(map[string]Node)
	byGroupKindNameNamespace := make(map[string]map[string]map[string]map[string]Node)

	for _, n := range nodes {
		byUID[n.UID] = n
		kind := n.Properties["kind"].(string)
		namespace := normalizeNamespace("")
		if ns, ok := n.Properties["namespace"].(string); ok {
			namespace = normalizeNamespace(ns)
		}
		name := n.Properties["name"].(string)
		group := ""
		if g, ok := n.Properties["apigroup"].(string); ok {
			group = g
		}

		if byGroupKindNameNamespace[group] == nil {
			byGroupKindNameNamespace[group] = map[string]map[string]map[string]Node{}
		}

		if byGroupKindNameNamespace[group][kind] == nil {
			byGroupKindNameNamespace[group][kind] = map[string]map[string]Node{}
		}

		if byGroupKindNameNamespace[group][kind][namespace] == nil {
			byGroupKindNameNamespace[group][kind][namespace] = map[string]Node{}
		}

		byGroupKindNameNamespace[group][kind][namespace][name] = n
	}

	store := NodeStore{
		ByUID:                    byUID,
		ByGroupKindNamespaceName: byGroupKindNameNamespace,
	}

	return store
}
