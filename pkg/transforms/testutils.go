// Contains utils for use in testing.

package transforms

import (
	"testing"
)

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
