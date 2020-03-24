/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package config keeps tracks of all configurations
package config

import (
	"errors"
	"flag"
	"os"
	"sync"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	configs map[string]*rest.Config
	lock    sync.Mutex
)

func init() {
	configs = make(map[string]*rest.Config)
}

// ParseFlagsOrDie creates config named "default" from flags and returns it or dies
func ParseFlagsOrDie() *rest.Config {
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}
	return config
}

// ParseFlags creates config named "default" from flags and returns it
func ParseFlags() (*rest.Config, error) {
	if flag.Lookup("master") == nil {
		flag.String("master", os.Getenv("KUBEMASTER"), "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	}
	if flag.Lookup("kubeconfig") == nil {
		flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "Path to a kube config. Only required if out-of-cluster.")
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	masterURL := flag.Lookup("master").Value.String()
	kubeconfig := flag.Lookup("kubeconfig").Value.String()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}

	Add("default", config)
	return config, nil
}

// Add keeps track of config and gives it a name (or "default" when empty).
func Add(name string, config *rest.Config) {
	lock.Lock()
	defer lock.Unlock()

	if name == "" {
		name = "default"
	}

	configs[name] = config
}

// Get returns the named config, or nil if none exist
func Get(name string) *rest.Config {
	if config, ok := configs[name]; ok {
		return config
	}
	return nil
}

// FindName returns the config name
func FindName(config *rest.Config) (string, error) {
	lock.Lock()
	defer lock.Unlock()

	for name, v := range configs {
		if v == config {
			return name, nil
		}
	}
	return "", errors.New("config not found")
}
