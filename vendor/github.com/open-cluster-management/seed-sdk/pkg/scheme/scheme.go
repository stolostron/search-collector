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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PluralScheme extends scheme with plurals
type PluralScheme struct {
	Scheme *runtime.Scheme

	// plurals for group, version and kinds
	plurals map[schema.GroupVersionKind]string
}

// NewScheme creates an object for given group/version/kind and set ObjectKind
func NewScheme() *PluralScheme {
	return &PluralScheme{Scheme: runtime.NewScheme(), plurals: make(map[schema.GroupVersionKind]string)}
}

// NewObject creates an object for given group/version/kind and set ObjectKind
func (p *PluralScheme) NewObject(gvk schema.GroupVersionKind) (runtime.Object, error) {
	obj, err := p.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	return obj, nil
}

// Plural returns the plural corresponding to the given group/version/kind
func (p *PluralScheme) Plural(gvk schema.GroupVersionKind) (string, error) {
	if plural, ok := p.plurals[gvk]; ok {
		return plural, nil
	}
	return "", fmt.Errorf("unrecognized group '%s' version '%s' and kind '%s'", gvk.Group, gvk.Version, gvk.Kind)
}

// SetPlural sets the plural for corresponding  group/version/kind
func (p *PluralScheme) SetPlural(gvk schema.GroupVersionKind, plural string) {
	p.plurals[gvk] = plural
}

// ResourceGroupVersionKind identifies a CRD.
type ResourceGroupVersionKind struct {
	Resource string // same as Plural
	Group    string
	Version  string
	Kind     string
}

// ToGroupVersion extract group and version
func (rgvk ResourceGroupVersionKind) ToGroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: rgvk.Group, Version: rgvk.Version}
}

// ToGroupVersionKind extract group, version and kind
func (rgvk ResourceGroupVersionKind) ToGroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: rgvk.Group, Version: rgvk.Version, Kind: rgvk.Kind}
}
