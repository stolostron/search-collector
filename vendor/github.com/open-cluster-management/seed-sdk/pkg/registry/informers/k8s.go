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

package informers

import (
	"fmt"
	"time"

	"github.com/open-cluster-management/seed-sdk/pkg/registry/clientsets"

	apiinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	k8sinternal "k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
)

// GetKube returns the Kubernetes SharedInformerFactory associated the given resource name or nil when none exist
// Supported name format are:
// [<informerName>/]<clientsetName>[@<configName>]
// where informerName default is  "default"
// and configName default is "default"
func GetKube(name string) k8sinformers.SharedInformerFactory {
	factory := Get(name)
	if factory == nil {
		return nil
	}
	if k8sfactory, ok := factory.(k8sinformers.SharedInformerFactory); ok {
		return k8sfactory
	}
	return nil
}

// APIExtensionsAPIServer returns the apiextensions-apiserver SharedInformerFactory associated the given resource name
// Supported name format are:
// [<informerName>/]<clientsetName>[@<configName>]
// where informerName default is  "default"
// and configName default is "default"
func APIExtensionsAPIServer(name string) (apiinformers.SharedInformerFactory, error) {
	factory := Get(name)
	if factory == nil {
		return nil, fmt.Errorf("the API extensions resource %s does not have a shared informer factory", name)
	}
	if k8sfactory, ok := factory.(apiinformers.SharedInformerFactory); ok {
		return k8sfactory, nil
	}
	return nil, fmt.Errorf("resource '%s' is not a API extensions resource", name)
}

// -- K8S constructor

func kubeProvider(clientset clientsets.Interface, defaultResync time.Duration, namespace string, listOptions func(*metav1.ListOptions)) SharedInformerFactory {
	tweak := k8sinternal.TweakListOptionsFunc(listOptions)
	return k8sinformers.NewFilteredSharedInformerFactory(clientset.(kubernetes.Interface), defaultResync, namespace, tweak).(SharedInformerFactory)
}

// AddKubeFactory registers Kubernetes informer factories
func AddKubeFactory(name string, clientset clientsets.Interface, defaultResync time.Duration) error {
	return AddFilteredKubeFactory(name, clientset, defaultResync, metav1.NamespaceAll, nil)
}

// AddFilteredKubeFactory registers Kubernetes informer factories
func AddFilteredKubeFactory(name string, clientset clientsets.Interface, defaultResync time.Duration, namespace string, listOptions func(*metav1.ListOptions)) error {
	return AddFilteredFactory(name, kubeProvider, clientset, defaultResync, namespace, listOptions)
}
