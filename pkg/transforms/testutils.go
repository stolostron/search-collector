// Contains utils for use in testing.

package transforms

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

// UnmarshalFile takes a file path and unmarshals it into the given resource type.
func UnmarshalFile(filepath string, resourceType interface{}, t *testing.T) {
	// open given filepath string
	rawBytes, err := ioutil.ReadFile(filepath)
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
