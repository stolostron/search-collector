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

package clientsets

import (
	"fmt"

	"github.com/open-cluster-management/seed-sdk/pkg/registry/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddKube registers k8s clientset for the given config name.
// The clientset name is "kubernetes@<name>"
func AddKube(configName string) (Interface, error) {
	cfg := config.Get(configName)
	if cfg == nil {
		return nil, fmt.Errorf("config %s does not exist", configName)
	}
	return AddKubeForConfig(cfg)
}

// AddKubeForConfig registers k8s clientset for the given config.
// The clientset name is "kube@<configName>"
func AddKubeForConfig(cfg *rest.Config) (Interface, error) {
	return AddClientsetForConfig("kubernetes", newKubeClientsetForConfig, cfg)
}

// AddKubeForConfigOrDie registers k8s clientset for the given config or die.
func AddKubeForConfigOrDie(config *rest.Config) Interface {
	clientset, err := AddKubeForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

// GetKube returns the Kubernetes clientset for the given configuration name, or nil if none exist
func GetKube(name string) kubernetes.Interface {
	clientset := Get("kubernetes@" + name)
	if clientset == nil {
		return nil
	}
	return clientset.(kubernetes.Interface)
}

func newKubeClientsetForConfig(cfg *rest.Config) (Interface, error) {
	return kubernetes.NewForConfig(cfg)
}
