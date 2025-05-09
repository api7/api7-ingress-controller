package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true

// PluginConfig is the Schema for the PluginConfigs API.
type PluginConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PluginConfigSpec `json:"spec,omitempty"`
}

// PluginConfigSpec defines the desired state of PluginConfig.
type PluginConfigSpec struct {
	// Plugins are an array of plugins and their configurations to be applied.
	Plugins []Plugin `json:"plugins"`
}

// +kubebuilder:object:root=true

// PluginConfigList contains a list of PluginConfig.
type PluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PluginConfig `json:"items"`
}

type Plugin struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`
	// Config is plugin configuration details.
	Config apiextensionsv1.JSON `json:"config,omitempty" yaml:"config,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PluginConfig{}, &PluginConfigList{})
}
