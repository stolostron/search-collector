// Copyright (c) 2021 Red Hat, Inc.

package config

import (
	"os"
	"testing"
)

// Should use default value when environment variable does not exist.
func Test_SetDefault_01(t *testing.T) {

	property := ""
	setDefault(&property, "ENV_VARIABLE_NOT_DEFINED", "default-value")

	if property != "default-value" {
		t.Errorf("Failed testing setDefault()  Expected: %s  Got: %s", "default-value", property)
	}
}

// Should use value from environment variable if it exists.
func Test_SetDefault_02(t *testing.T) {

	os.Setenv("TEST_ENV_VARIABLE", "value-from-env")
	property := ""
	setDefault(&property, "TEST_ENV_VARIABLE", "default-value")

	if property != "value-from-env" {
		t.Errorf("Failed testing setDefault()  Expected: %s  Got: %s", "value-from-env", property)
	}
}

func Test_SetDefaultInt_01(t *testing.T) {

	var property int
	setDefaultInt(&property, "ENV_VARIABLE_NOT_DEFINED", 1234)

	if property != 1234 {
		t.Errorf("Failed testing setDefault()  Expected: %d  Got: %d", 1234, property)
	}
}

// Should use value from environment variable if it exists.
func Test_SetDefaultInt_02(t *testing.T) {

	os.Setenv("TEST_ENV_VARIABLE", "9999")
	var property int
	setDefaultInt(&property, "TEST_ENV_VARIABLE", 0000)

	if property != 9999 {
		t.Errorf("Failed testing setDefault()  Expected: %d  Got: %d", 9999, property)
	}
}
