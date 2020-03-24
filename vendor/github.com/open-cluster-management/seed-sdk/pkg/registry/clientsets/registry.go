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

// Package clientsets keeps tracks of all clientsets.
package clientsets

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/open-cluster-management/seed-sdk/pkg/registry/config"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// Interface is the base interface for all typed clientset
type Interface interface {
	Discovery() discovery.DiscoveryInterface
}

// ClientsetConstructor allows creating Clientset from base clientset type (Interface)
type ClientsetConstructor func(cfg *rest.Config) (Interface, error)

var (
	clientsets map[string]Interface
	lock       sync.Mutex
)

func init() {
	clientsets = make(map[string]Interface)
}

// AddClientsetForConfig registers the clientset returned by constructor
// The clientset is registered under the fullname <name>@<configName>
func AddClientsetForConfig(name string, constructor ClientsetConstructor, cfg *rest.Config) (Interface, error) {
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("invalide clientset name %s (cannot contains '/')", name)
	}
	configName, err := config.FindName(cfg)
	if err != nil {
		return nil, err
	}
	clientset, err := constructor(cfg)
	if err != nil {
		return nil, err
	}
	Add(name+"@"+configName, clientset)
	return clientset, nil
}

// AddClientsetForConfigOrDie registers the clientset obtained by constructor. Panic when an error occurs
func AddClientsetForConfigOrDie(name string, constructor ClientsetConstructor, cfg *rest.Config) Interface {
	clientset, err := AddClientsetForConfig(name, constructor, cfg)
	if err != nil {
		panic(err)
	}
	return clientset
}

// Add keeps track of clientset and gives it a name.
func Add(name string, clientset Interface) {
	lock.Lock()
	defer lock.Unlock()

	clientsets[name] = clientset
}

// Get returns the clientset named name, or nil if none exist
//
// Supported formats:
// <clientsetName>[@<configName>]
// where configName default is "default"
func Get(name string) Interface {
	lock.Lock()
	defer lock.Unlock()

	if strings.Index(name, "@") == -1 {
		name = name + "@default"
	}

	if clientset, ok := clientsets[name]; ok {
		return clientset
	}
	return nil
}

// FindName returns the clientset name
func FindName(clientset Interface) (string, error) {
	lock.Lock()
	defer lock.Unlock()

	for name, v := range clientsets {
		if v == clientset {
			return name, nil
		}
	}
	return "", errors.New("clientset not found")
}

// FindConfigName returns the configuration name bound to clientset
func FindConfigName(clientset Interface) (string, error) {
	name, err := FindName(clientset)
	if err != nil {
		return "", err
	}

	if i := strings.Index(name, "@"); i != -1 {
		return name[i+1:], nil
	}
	return "", fmt.Errorf("clientset name '%s' malformed: missing '@<configName>' suffix", name)
}
