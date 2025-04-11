/*
Copyright 2024.

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

package v1alpha1

import (
	"encoding/json"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// HTTPRoutePolicySpec defines the desired state of HTTPRoutePolicy.
type HTTPRoutePolicySpec struct {
	// TargetRef identifies an API object (enum: HTTPRoute, Ingress) to apply HTTPRoutePolicy to.
	//
	// target references.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []gatewayv1alpha2.LocalPolicyTargetReferenceWithSectionName `json:"targetRefs"`

	Policy HTTPRoutePolicySpecPolicy `json:"policy"`
}

// +kubebuilder:object:root=true

// HTTPRoutePolicy is the Schema for the httproutepolicies API.
type HTTPRoutePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HTTPRoutePolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// HTTPRoutePolicyList contains a list of HTTPRoutePolicy.
type HTTPRoutePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPRoutePolicy `json:"items"`
}

type HTTPRoutePolicySpecPolicy struct {
	Priority *int64 `json:"priority,omitempty" yaml:"priority,omitempty"`
	Vars     Vars   `json:"vars,omitempty" yaml:"vars,omitempty"`
}

// Vars represents the route match expressions of APISIX.
type Vars [][]StringOrSlice

// UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (vars *Vars) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var data [][]StringOrSlice
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*vars = data
	return nil
}

// StringOrSlice represents a string or a string slice.
// TODO Do not use interface{} to avoid the reflection overheads.
type StringOrSlice struct {
	StrVal   string   `json:"-"`
	SliceVal []string `json:"-"`
}

func (s *StringOrSlice) MarshalJSON() ([]byte, error) {
	var (
		p   []byte
		err error
	)
	if s.SliceVal != nil {
		p, err = json.Marshal(s.SliceVal)
	} else {
		p, err = json.Marshal(s.StrVal)
	}
	return p, err
}

func (s *StringOrSlice) UnmarshalJSON(p []byte) error {
	var err error

	if len(p) == 0 {
		return errors.New("empty object")
	}
	if p[0] == '[' {
		err = json.Unmarshal(p, &s.SliceVal)
	} else {
		err = json.Unmarshal(p, &s.StrVal)
	}
	return err
}

func init() {
	SchemeBuilder.Register(&HTTPRoutePolicy{}, &HTTPRoutePolicyList{})
}
