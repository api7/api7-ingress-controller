package translator

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

type Translator struct {
	Log logr.Logger
}

type TranslateContext struct {
	BackendRefs      []gatewayv1.BackendRef
	EndpointSlices   map[types.NamespacedName][]discoveryv1.EndpointSlice
	GatewayTLSConfig []gatewayv1.GatewayTLSConfig
	Secrets          map[types.NamespacedName]*corev1.Secret
}

type TranslateResult struct {
	Routes   []*v1.Route
	Services []*v1.Service
	SSL      []*v1.Ssl
}
