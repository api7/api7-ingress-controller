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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/api7/api7-ingress-controller/api/common"
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

	Spec   HTTPRoutePolicySpec          `json:"spec,omitempty"`
	Status gatewayv1alpha2.PolicyStatus `json:"status,omitempty"`
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
// +kubebuilder:object:generate=false
type Vars = common.Vars

func init() {
	SchemeBuilder.Register(&HTTPRoutePolicy{}, &HTTPRoutePolicyList{})
}
