/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package informers keeps track of all SharedInformer and GenericLister
package informers

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/seed-sdk/pkg/registry/clientsets"
)

// SharedInformerFactory provides base interface for shared informers factories
type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
}

// SharedInformerFactoryConstructor allows constructing SharedInformerFactory from generic types
type SharedInformerFactoryConstructor func(clientset clientsets.Interface, defaultResync time.Duration, namespace string, listOptions func(*metav1.ListOptions)) SharedInformerFactory

var (
	factories map[string]SharedInformerFactory
	lock      sync.Mutex
)

func init() {
	factories = make(map[string]SharedInformerFactory)
}

// AddFactory registers the SharedInformerFactory returned by constructor
// The factory is registered under the fullname <name>/<clientsetName>@<configName>
func AddFactory(name string, constructor SharedInformerFactoryConstructor, clientset clientsets.Interface, defaultResync time.Duration) error {
	return AddFilteredFactory(name, constructor, clientset, defaultResync, metav1.NamespaceAll, nil)
}

// AddFilteredFactory registers informer factories returned by constructor
// The factory is registered under the fullname <name>/<clientsetName>@<configName>
func AddFilteredFactory(name string, constructor SharedInformerFactoryConstructor, clientset clientsets.Interface, defaultResync time.Duration, namespace string, listOptions func(*metav1.ListOptions)) error {
	if strings.Contains(name, "/") {
		return fmt.Errorf("invalid informer name %s (cannot contains '/')", name)
	}

	factory := constructor(clientset, defaultResync, namespace, listOptions)
	if factory == nil {
		return fmt.Errorf("invalid clientset %v ", reflect.TypeOf(clientset))
	}

	csName, err := clientsets.FindName(clientset)
	if err != nil {
		return err
	}

	Add(name+"/"+csName, factory)
	return nil
}

// Add ...
func Add(name string, factory SharedInformerFactory) {
	lock.Lock()
	defer lock.Unlock()

	factories[name] = factory
}

// Get returns the SharedInformerFactory of the given name or nil when none exist
//
// Supported name format are:
// [<informerName>/]<clientsetName>[@<configName>]
// where informerName default is  "default"
// and configName default is "default"
func Get(name string) SharedInformerFactory {
	if factory, ok := factories[fix(name)]; ok {
		return factory
	}
	return nil
}

func fix(name string) string {
	if strings.Index(name, "@") == -1 {
		name = name + "@default"
	}
	if strings.Index(name, "/") == -1 {
		name = "default/" + name
	}
	return name
}
