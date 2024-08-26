package translator

import (
	"github.com/go-logr/logr"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

type Translator struct {
	Log logr.Logger
}

type TranslateContext struct {
	Gateways       []*gatewayv1.Gateway
	BackendRefs    []gatewayv1.BackendRef
	EndpointSlices map[types.NamespacedName][]discoveryv1.EndpointSlice
}

type TranslateResult struct {
	Routes   []*v1.Route
	Services []*v1.Service
}
