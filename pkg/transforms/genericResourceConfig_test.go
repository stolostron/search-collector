// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStringToDataType tests the string→DataType mapping helper.
func TestStringToDataType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DataType
	}{
		{"bytes", "bytes", DataTypeBytes},
		{"slice", "slice", DataTypeSlice},
		{"string", "string", DataTypeString},
		{"number", "number", DataTypeNumber},
		{"mapString", "mapString", DataTypeMapString},
		{"Empty String", "", DataTypeString},             // Default
		{"Unknown Value", "UnknownType", DataTypeString}, // Default
		{"Invalid Case", "Bytes", DataTypeString},        // Case-sensitive, should default
		{"Random String", "foobar", DataTypeString},      // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToDataType(tt.input)
			assert.Equal(t, tt.expected, result, "DataType mismatch for input: %s", tt.input)
		})
	}
}
