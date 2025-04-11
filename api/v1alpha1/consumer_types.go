package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConsumerSpec `json:"spec,omitempty"`
	Status Status       `json:"status,omitempty"`
}

type ConsumerSpec struct {
	GatewayRef  GatewayRef   `json:"gatewayRef,omitempty"`
	Credentials []Credential `json:"credentials,omitempty"`
	Plugins     []Plugin     `json:"plugins,omitempty"`
}

type GatewayRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:default=Gateway
	Kind *string `json:"kind,omitempty"`
	// +kubebuilder:default=gateway.networking.k8s.io
	Group     *string `json:"group,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type Credential struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=jwt-auth;basic-auth;key-auth;hmac-auth;
	Type      string               `json:"type"`
	Config    apiextensionsv1.JSON `json:"config,omitempty"`
	SecretRef *SecretReference     `json:"secretRef,omitempty"`
	Name      string               `json:"name,omitempty"`
}

type SecretReference struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Consumer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Consumer{}, &ConsumerList{})
}
