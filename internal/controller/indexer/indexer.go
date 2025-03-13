package indexer

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	ServiceIndexRef = "serviceRef"
	ExtensionRef    = "extensionRef"
	ParametersRef   = "parametersRef"
)

func GenIndexKey(namespace, name string) string {
	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}.String()
}

func HTTPRouteServiceIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	var keys []string
	for _, rule := range hr.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			namespace := hr.GetNamespace()
			if backend.Kind != nil && *backend.Kind != "Service" {
				continue
			}
			if backend.Namespace != nil {
				namespace = string(*backend.Namespace)
			}
			keys = append(keys, GenIndexKey(namespace, string(backend.Name)))
		}
	}
	return keys
}

func HTTPRouteExtensionIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	var keys []string
	for _, rule := range hr.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				keys = append(keys, GenIndexKey(hr.GetNamespace(), string(filter.ExtensionRef.Name)))
			}
		}
	}
	return keys
}

func GatewayParametersRefIndexFunc(rawObj client.Object) []string {
	gw := rawObj.(*gatewayv1.Gateway)
	var keys []string
	if gw.Spec.Infrastructure != nil && gw.Spec.Infrastructure.ParametersRef != nil {
		// now we only care about kind: GatewayProxy
		if gw.Spec.Infrastructure.ParametersRef.Kind == "GatewayProxy" {
			keys = append(keys, GenIndexKey(gw.GetNamespace(), string(gw.Spec.Infrastructure.ParametersRef.Name)))
		}
	}
	return keys
}
