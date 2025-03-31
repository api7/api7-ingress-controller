package indexer

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	ServiceIndexRef = "serviceRefs"
	ExtensionRef    = "extensionRef"
	ParametersRef   = "parametersRef"
	ParentRefs      = "parentRefs"
	SecretIndexRef  = "secretRefs"
	IngressClass    = "ingressClass"
	IngressClassRef = "ingressClassRef"
)

func SetupIndexer(mgr ctrl.Manager) error {
	if err := setupGatewayIndexer(mgr); err != nil {
		return err
	}
	if err := setupHTTPRouteIndexer(mgr); err != nil {
		return err
	}
	if err := setupIngressIndexer(mgr); err != nil {
		return err
	}
	return nil
}

func setupGatewayIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.Gateway{},
		ParametersRef,
		GatewayParametersRefIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func setupHTTPRouteIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ParentRefs,
		HTTPRouteParentRefsIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ExtensionRef,
		HTTPRouteExtensionIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ServiceIndexRef,
		HTTPRouteServiceIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func setupIngressIndexer(mgr ctrl.Manager) error {
	// create IngressClass index
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		IngressClassRef,
		IngressClassRefIndexFunc,
	); err != nil {
		return err
	}

	// create Service index for quick lookup of Ingresses using specific services
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		ServiceIndexRef,
		IngressServiceIndexFunc,
	); err != nil {
		return err
	}

	// create secret index for TLS
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		SecretIndexRef,
		IngressSecretIndexFunc,
	); err != nil {
		return err
	}

	// create IngressClass index
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.IngressClass{},
		IngressClass,
		IngressClassIndexFunc,
	); err != nil {
		return err
	}

	return nil
}

func IngressClassIndexFunc(rawObj client.Object) []string {
	ingressClass := rawObj.(*networkingv1.IngressClass)
	if ingressClass.Spec.Controller == "" {
		return nil
	}
	controllerName := ingressClass.Spec.Controller
	return []string{controllerName}
}

func IngressClassRefIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	if ingress.Spec.IngressClassName == nil {
		return nil
	}
	return []string{*ingress.Spec.IngressClassName}
}

func IngressServiceIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	var services []string

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			key := GenIndexKey(ingress.Namespace, path.Backend.Service.Name)
			services = append(services, key)
		}
	}
	return services
}

func IngressSecretIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	secrets := make([]string, 0)

	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		key := GenIndexKey(ingress.Namespace, tls.SecretName)
		secrets = append(secrets, key)
	}
	return secrets
}

func GenIndexKey(namespace, name string) string {
	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}.String()
}

func HTTPRouteParentRefsIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	keys := make([]string, 0, len(hr.Spec.ParentRefs))
	for _, ref := range hr.Spec.ParentRefs {
		ns := hr.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		keys = append(keys, GenIndexKey(ns, string(ref.Name)))
	}
	return keys
}

func HTTPRouteServiceIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	keys := make([]string, 0, len(hr.Spec.Rules))
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
	keys := make([]string, 0, len(hr.Spec.Rules))
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
	if gw.Spec.Infrastructure != nil && gw.Spec.Infrastructure.ParametersRef != nil {
		// now we only care about kind: GatewayProxy
		if gw.Spec.Infrastructure.ParametersRef.Kind == "GatewayProxy" {
			name := gw.Spec.Infrastructure.ParametersRef.Name
			return []string{GenIndexKey(gw.GetNamespace(), name)}
		}
	}
	return nil
}
