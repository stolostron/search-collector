// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Should use default value when environment variable does not exist.
func Test_SetDefault_01(t *testing.T) {

	property := ""
	setDefault(&property, "ENV_VARIABLE_NOT_DEFINED", "default-value")

	if property != "default-value" {
		t.Errorf("Failed testing setDefault() Expected: %s  Got: %s", "default-value", property)
	}
}

func TestInitConfigValidationRate(t *testing.T) {
	oldCfg := Cfg
	Cfg = Config{}
	defer func() { Cfg = oldCfg }()

	t.Setenv("DEPLOYED_IN_HUB", "true")
	t.Setenv("VALIDATION_RATE_MS", "12345")

	InitConfig()

	assert.Equal(t, 12345, Cfg.ValidationRateMS)
}

func TestInitConfigValidationRateDefault(t *testing.T) {
	oldCfg := Cfg
	Cfg = Config{}
	defer func() { Cfg = oldCfg }()

	t.Setenv("DEPLOYED_IN_HUB", "true")
	t.Setenv("VALIDATION_RATE_MS", "")

	InitConfig()

	assert.Equal(t, DEFAULT_VALIDATION_RATE_MS, Cfg.ValidationRateMS)
}

func TestInitConfigValidationMaxPending(t *testing.T) {
	for _, test := range []struct {
		name     string
		value    string
		expected int
	}{
		{name: "default", expected: 3600000},
		{name: "override", value: "120000", expected: 120000},
		{name: "zero", value: "0", expected: 3600000},
		{name: "negative", value: "-1", expected: 3600000},
		{name: "invalid", value: "invalid", expected: 3600000},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldCfg := Cfg
			Cfg = Config{}
			defer func() { Cfg = oldCfg }()
			t.Setenv("DEPLOYED_IN_HUB", "true")
			t.Setenv("VALIDATION_MAX_PENDING_MS", test.value)

			InitConfig()

			assert.Equal(t, test.expected, Cfg.ValidationMaxPendingMS)
		})
	}
}

// Should use value from environment variable if it exists.
func Test_SetDefault_02(t *testing.T) {

	_ = os.Setenv("TEST_ENV_VARIABLE", "value-from-env")
	property := ""
	setDefault(&property, "TEST_ENV_VARIABLE", "default-value")

	if property != "value-from-env" {
		t.Errorf("Failed testing setDefault() Expected: %s  Got: %s", "value-from-env", property)
	}
}

func Test_SetDefaultInt_01(t *testing.T) {

	var property int
	setDefaultInt(&property, "ENV_VARIABLE_NOT_DEFINED", 1234)

	if property != 1234 {
		t.Errorf("Failed testing setDefault() Expected: %d  Got: %d", 1234, property)
	}
}

// Should use value from environment variable if it exists.
func Test_SetDefaultInt_02(t *testing.T) {

	_ = os.Setenv("TEST_ENV_VARIABLE", "9999")
	var property int
	setDefaultInt(&property, "TEST_ENV_VARIABLE", 0000)

	if property != 9999 {
		t.Errorf("Failed testing setDefault() Expected: %d  Got: %d", 9999, property)
	}
}

func Test_InitConfig_ConfigurableCollectionDefaultsOn(t *testing.T) {
	origCfg := Cfg
	t.Cleanup(func() { Cfg = origCfg })

	t.Setenv("DEPLOYED_IN_HUB", "true")
	t.Setenv("FEATURE_CONFIGURABLE_COLLECTION", "")
	Cfg.FeatureConfigurableCollection = false

	InitConfig()

	if !Cfg.FeatureConfigurableCollection {
		t.Error("expected configurable collection to be enabled by default")
	}
}

func Test_InitConfig_ConfigurableCollectionKillSwitch(t *testing.T) {
	origCfg := Cfg
	t.Cleanup(func() { Cfg = origCfg })

	t.Setenv("DEPLOYED_IN_HUB", "true")
	t.Setenv("FEATURE_CONFIGURABLE_COLLECTION", "false")
	Cfg.FeatureConfigurableCollection = true

	InitConfig()

	if Cfg.FeatureConfigurableCollection {
		t.Error("expected configurable collection to be disabled when explicitly set to false")
	}
}
