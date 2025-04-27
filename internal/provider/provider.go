package provider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
)

type Provider interface {
	Update(context.Context, *TranslateContext, client.Object) error
	Delete(context.Context, client.Object) error
}

type ResourceKind struct {
	Kind      string
	Namespace string
	Name      string
}

type TranslateContext struct {
	context.Context
	RouteParentRefs  []gatewayv1.ParentReference
	BackendRefs      []gatewayv1.BackendRef
	GatewayTLSConfig []gatewayv1.GatewayTLSConfig
	Credentials      []v1alpha1.Credential

	EndpointSlices         map[types.NamespacedName][]discoveryv1.EndpointSlice
	Secrets                map[types.NamespacedName]*corev1.Secret
	PluginConfigs          map[types.NamespacedName]*v1alpha1.PluginConfig
	Services               map[types.NamespacedName]*corev1.Service
	BackendTrafficPolicies map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy
	GatewayProxies         map[ResourceKind]v1alpha1.GatewayProxy
	ResourceParentRefs     map[ResourceKind][]ResourceKind
	HTTPRoutePolicies      []v1alpha1.HTTPRoutePolicy

	StatusUpdaters []client.Object
}

func NewDefaultTranslateContext(ctx context.Context) *TranslateContext {
	return &TranslateContext{
		Context:                ctx,
		EndpointSlices:         make(map[types.NamespacedName][]discoveryv1.EndpointSlice),
		Secrets:                make(map[types.NamespacedName]*corev1.Secret),
		PluginConfigs:          make(map[types.NamespacedName]*v1alpha1.PluginConfig),
		Services:               make(map[types.NamespacedName]*corev1.Service),
		BackendTrafficPolicies: make(map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy),
		GatewayProxies:         make(map[ResourceKind]v1alpha1.GatewayProxy),
		ResourceParentRefs:     make(map[ResourceKind][]ResourceKind),
	}
}
