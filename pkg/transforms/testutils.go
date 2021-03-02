/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/
// Copyright (c) 2020 Red Hat, Inc.

// Contains utils for use in testing.
package transforms

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	sanitize "github.com/kennygrant/sanitize"
)

// UnmarshalFile takes a file path and unmarshals it into the given resource type.
func UnmarshalFile(filepath string, resourceType interface{}, t *testing.T) {
	// open given filepath string
	rawBytes, err := ioutil.ReadFile("../../test-data/" + sanitize.Name(filepath))
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
	byKindNameNamespace := make(map[string]map[string]map[string]Node)

	for _, n := range nodes {
		byUID[n.UID] = n
		kind := n.Properties["kind"].(string)
		var namespace = "_NONE"
		if n.Properties["namespace"] != nil {
			namespace = n.Properties["namespace"].(string)
		}

		byKindNameNamespace[kind] = make(map[string]map[string]Node)
		byKindNameNamespace[kind][namespace] = make(map[string]Node)
		byKindNameNamespace[kind][namespace][n.Properties["name"].(string)] = n
	}

	store := NodeStore{
		ByUID:               byUID,
		ByKindNamespaceName: byKindNameNamespace,
	}

	return store
}
