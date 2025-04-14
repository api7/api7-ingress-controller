package provider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider interface {
	Update(context.Context, *TranslateContext, client.Object) error
	Delete(context.Context, client.Object) error
}

type TranslateContext struct {
	BackendRefs      []gatewayv1.BackendRef
	GatewayTLSConfig []gatewayv1.GatewayTLSConfig
	GatewayProxies   []v1alpha1.GatewayProxy
	Credentials      []v1alpha1.Credential
	EndpointSlices   map[types.NamespacedName][]discoveryv1.EndpointSlice
	Secrets          map[types.NamespacedName]*corev1.Secret
	PluginConfigs    map[types.NamespacedName]*v1alpha1.PluginConfig
	Services         map[types.NamespacedName]*corev1.Service
}

func NewDefaultTranslateContext() *TranslateContext {
	return &TranslateContext{
		EndpointSlices: make(map[types.NamespacedName][]discoveryv1.EndpointSlice),
		Secrets:        make(map[types.NamespacedName]*corev1.Secret),
		PluginConfigs:  make(map[types.NamespacedName]*v1alpha1.PluginConfig),
		Services:       make(map[types.NamespacedName]*corev1.Service),
	}
}
