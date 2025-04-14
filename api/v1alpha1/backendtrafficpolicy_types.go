package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type BackendTrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackendTrafficPolicySpec `json:"spec,omitempty"`
	Status PolicyStatus             `json:"status,omitempty"`
}

type BackendTrafficPolicySpec struct {
	// TargetRef identifies an API object to apply policy to.
	// Currently, Backends (i.e. Service, ServiceImport, or any
	// implementation-specific backendRef) are the only valid API
	// target references.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []BackendPolicyTargetReferenceWithSectionName `json:"targetRefs"`
	// LoadBalancer represents the load balancer configuration for Kubernetes Service.
	// The default strategy is round robin.
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty" yaml:"loadbalancer,omitempty"`
	// The scheme used to talk with the upstream.
	//
	// +kubebuilder:validation:Enum=http;https;grpc;grpcs;
	// +kubebuilder:default=http
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// How many times that the proxy (Apache APISIX) should do when
	// errors occur (error, timeout or bad http status codes like 500, 502).
	// +optional
	Retries *int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout settings for the read, send and connect to the upstream.
	Timeout *Timeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Configures the host when the request is forwarded to the upstream.
	// Can be one of pass, node or rewrite.
	//
	// +kubebuilder:validation:Enum=pass;node;rewrite;
	// +kubebuilder:default=pass
	PassHost string `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`

	// Specifies the host of the Upstream request. This is only valid if
	// the pass_host is set to rewrite
	Host Hostname `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`
}

// LoadBalancer describes the load balancing parameters.
// +kubebuilder:validation:XValidation:rule="!(has(self.key) && self.type != 'chash')"
type LoadBalancer struct {
	// +kubebuilder:validation:Enum=roundrobin;chash;ewma;least_conn;
	// +kubebuilder:default=roundrobin
	// +kubebuilder:validation:Required
	Type string `json:"type" yaml:"type"`
	// The HashOn and Key fields are required when Type is "chash".
	// HashOn represents the key fetching scope.
	// +kubebuilder:validation:Enum=vars;header;cookie;consumer;vars_combinations;
	// +kubebuilder:default=vars
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// Key represents the hash key.
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
}

type Timeout struct {
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	Send    metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	Read    metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// +kubebuilder:object:root=true
type BackendTrafficPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackendTrafficPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackendTrafficPolicy{}, &BackendTrafficPolicyList{})
}
