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
	"reflect"
)

// Start initializes all non-running informers. Safe to call multiple times
func Start(stopCh <-chan struct{}) {
	lock.Lock()
	defer lock.Unlock()

	for _, factory := range factories {
		factory.Start(stopCh)
	}
}

// WaitForCacheSync waits for all started informers' cache were synced.
func WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool {
	lock.Lock()
	defer lock.Unlock()

	res := map[reflect.Type]bool{}
	for _, factory := range factories {
		synced := factory.WaitForCacheSync(stopCh)
		for k, v := range synced {
			res[k] = v
		}
	}
	return res
}
