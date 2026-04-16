// Copyright Contributors to the Open Cluster Management project

//go:build development
// +build development

// This file is excluded from compilation unless the build flag -tags development is used.
// Use `make run` to run with the development flag.
package config

import (
	"os"

	"k8s.io/klog/v2"
)

func init() {
	klog.Warning("!!! Running in development mode. !!!")
	os.Setenv("FEATURE_CONFIGURABLE_COLLECTION", "true")
	Cfg = new()
}
