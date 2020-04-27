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

package scheme

import (
	"github.com/open-cluster-management/seed-sdk/pkg/registry/clientsets"
	"github.com/open-cluster-management/seed-sdk/pkg/registry/informers"
	"k8s.io/apimachinery/pkg/runtime"
)

// CRDsInfo stores metadata about CRDs
type CRDsInfo struct {
	// SchemeBuilders associated to the CRDs
	SchemeBuilder runtime.SchemeBuilder

	// Unique clientset name
	// Typically <group>.<version>
	// For multiple versions: <group>
	// For multiple groups/versions: <your_custom_clientname>
	ClientsetName string

	// Typed clientset constructor
	ClientsetConstructor clientsets.ClientsetConstructor

	// Typed informer factory constructor
	InformerFactoryConstructor informers.SharedInformerFactoryConstructor

	// CRDs indexed by name
	CRDs map[string]CRDInfo
}

// CRDInfo stores metadata about a CRD
type CRDInfo struct {
	// Name is the CRD name, eg. functions.openwhisk.ibm.com
	Name string

	// RawCRD represents the raw YAML CRD
	RawCRD string
}
