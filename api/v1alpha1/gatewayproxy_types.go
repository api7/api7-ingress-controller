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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GatewayProxySpec defines the desired state of GatewayProxy
type GatewayProxySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Plugins        []GatewayProxyPlugin            `json:"plugins,omitempty"`
	PluginMetadata map[string]apiextensionsv1.JSON `json:"pluginMetadata,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayProxy is the Schema for the gatewayproxies API
type GatewayProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GatewayProxySpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayProxyList contains a list of GatewayProxy
type GatewayProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayProxy `json:"items"`
}

type GatewayProxyPlugin struct {
	Name    string               `json:"name,omitempty"`
	Enabled bool                 `json:"enabled,omitempty"`
	Config  apiextensionsv1.JSON `json:"config,omitempty"`
}

func init() {
	SchemeBuilder.Register(&GatewayProxy{}, &GatewayProxyList{})
}
