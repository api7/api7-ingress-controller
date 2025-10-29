// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// IngressReconciler reconciles a Ingress object.
type IngressReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider     provider.Provider
	genericEvent chan event.GenericEvent

	Updater status.Updater
	Readier readiness.ReadinessManager

	// supportsEndpointSlice indicates whether the cluster supports EndpointSlice API
	supportsEndpointSlice bool
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.genericEvent = make(chan event.GenericEvent, 100)

	// Check and store EndpointSlice API support
	r.supportsEndpointSlice = pkgutils.HasAPIResource(mgr, &discoveryv1.EndpointSlice{})

	eventFilters := []predicate.Predicate{
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
		predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
	}

	if !r.supportsEndpointSlice {
		eventFilters = append(eventFilters, predicate.NewPredicateFuncs(TypePredicate[*corev1.Endpoints]()))
	}

	bdr := ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{},
			builder.WithPredicates(
				MatchesIngressClassPredicate(r.Client, r.Log, ""),
			),
		).
		WithEventFilter(predicate.Or(eventFilters...)).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		)

	// Conditionally watch EndpointSlice or Endpoints based on cluster API support
	bdr = watchEndpointSliceOrEndpoints(bdr, r.supportsEndpointSlice,
		r.listIngressesByService,
		r.listIngressesByEndpoints,
		r.Log)

	return bdr.
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesBySecret),
		).
		Watches(&v1alpha1.BackendTrafficPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressForBackendTrafficPolicy),
			builder.WithPredicates(
				BackendTrafficPolicyPredicateFunc(r.genericEvent),
			),
		).
		Watches(&v1alpha1.HTTPRoutePolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesByHTTPRoutePolicy),
			builder.WithPredicates(httpRoutePolicyPredicateFuncs(r.genericEvent)),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesForGatewayProxy),
		).
		Watches(&apiv2.ApisixPluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressesForPluginConfig),
		).
		WatchesRawSource(
			source.Channel(
				r.genericEvent,
				handler.EnqueueRequestsFromMapFunc(r.listIngressForGenericEvent),
			),
		).
		Complete(r)
}

// Reconcile handles the reconciliation of Ingress resources
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&networkingv1.Ingress{}, req.NamespacedName)
	ingress := new(networkingv1.Ingress)
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		if client.IgnoreNotFound(err) == nil {
			if err := r.updateHTTPRoutePolicyStatusOnDeleting(ctx, req.NamespacedName); err != nil {
				return ctrl.Result{}, err
			}

			// Ingress was deleted, clean up corresponding resources
			ingress.Namespace = req.Namespace
			ingress.Name = req.Name

			ingress.TypeMeta = metav1.TypeMeta{
				Kind:       KindIngress,
				APIVersion: networkingv1.SchemeGroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, ingress); err != nil {
				r.Log.Error(err, "failed to delete ingress resources", "ingress", ingress.Name)
				return ctrl.Result{}, err
			}
			r.Log.Info("deleted ingress resources", "ingress", ingress.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reconciling ingress", "ingress", ingress.Name)

	// create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := FindMatchingIngressClassByObject(tctx, r.Client, r.Log, ingress, "")
	if err != nil {
		if err := r.Provider.Delete(ctx, ingress); err != nil {
			r.Log.Error(err, "failed to delete ingress resources", "ingress", ingress.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	tctx.RouteParentRefs = append(tctx.RouteParentRefs, gatewayv1.ParentReference{
		Group: ptr.To(gatewayv1.Group(ingressClass.GroupVersionKind().Group)),
		Kind:  ptr.To(gatewayv1.Kind(KindIngressClass)),
		Name:  gatewayv1.ObjectName(ingressClass.Name),
	})

	// process IngressClass parameters if they reference GatewayProxy
	if err := ProcessIngressClassParameters(tctx, r.Client, r.Log, ingress, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, err
	}

	// process TLS configuration
	if err := r.processTLS(tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process TLS configuration", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// process backend services
	if err := r.processBackends(tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process backend services", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// process plugin config annotation
	if err := r.processPluginConfig(tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process PluginConfig annotation", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// process HTTPRoutePolicy
	if err := r.processHTTPRoutePolicies(tctx, ingress); err != nil {
		r.Log.Error(err, "failed to process HTTPRoutePolicy", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)

	// update the ingress resources
	if err := r.Provider.Update(ctx, tctx, ingress); err != nil {
		r.Log.Error(err, "failed to update ingress resources", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	// update the status of related resources
	UpdateStatus(r.Updater, r.Log, tctx)

	// update the ingress status
	if err := r.updateStatus(ctx, tctx, ingress, ingressClass); err != nil {
		r.Log.Error(err, "failed to update ingress status", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// matchesIngressController check if the ingress class is controlled by us
func (r *IngressReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return false
	}

	return matchesController(ingressClass.Spec.Controller)
}

// listIngressForIngressClass list all ingresses that use a specific ingress class
func (r *IngressReconciler) listIngressForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return nil
	}

	// check if the ingress class is the default ingress class
	if IsDefaultIngressClass(ingressClass) {
		ingressList := &networkingv1.IngressList{}
		if err := r.List(ctx, ingressList); err != nil {
			r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
			return nil
		}

		requests := make([]reconcile.Request, 0, len(ingressList.Items))
		for i := range ingressList.Items {
			ingress := &ingressList.Items[i]
			effectiveClassName := internaltypes.GetEffectiveIngressClassName(ingress)
			if effectiveClassName == "" || effectiveClassName == ingressClass.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: ingress.GetNamespace(),
						Name:      ingress.GetName(),
					},
				})
			}
		}
		return requests
	} else {
		ingressList := &networkingv1.IngressList{}
		if err := r.List(ctx, ingressList, client.MatchingFields{
			indexer.IngressClassRef: ingressClass.GetName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
			return nil
		}

		requests := make([]reconcile.Request, 0, len(ingressList.Items))
		for _, ingress := range ingressList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}

		return requests
	}
}

// listIngressesByService list all ingresses that use a specific service
func (r *IngressReconciler) listIngressesByService(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}

	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by service", "service", serviceName)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, "") {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}
	}
	return requests
}

// listIngressesByEndpoints handles Endpoints objects and converts them to Ingress reconcile requests.
// This function provides backward compatibility for Kubernetes 1.18 clusters that don't support EndpointSlice.
func (r *IngressReconciler) listIngressesByEndpoints(ctx context.Context, obj client.Object) []reconcile.Request {
	endpoint, ok := obj.(*corev1.Endpoints)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Endpoints")
		return nil
	}

	namespace := endpoint.GetNamespace()
	serviceName := endpoint.GetName() // For Endpoints, the name is the service name

	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by service", "service", serviceName)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, "") {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}
	}
	return requests
}

// listIngressesBySecret list all ingresses that use a specific secret
func (r *IngressReconciler) listIngressesBySecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Secret")
		return nil
	}

	namespace := secret.GetNamespace()
	name := secret.GetName()

	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by secret", "secret", name)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, "") {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ingress.Namespace,
					Name:      ingress.Name,
				},
			})
		}
	}

	gatewayProxyList := &v1alpha1.GatewayProxyList{}
	if err := r.List(ctx, gatewayProxyList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list gateway proxies by secret", "secret", secret.GetName())
		return nil
	}

	for _, gatewayProxy := range gatewayProxyList.Items {
		var (
			ingressClassList networkingv1.IngressClassList
			indexKey         = indexer.GenIndexKey(gatewayProxy.GetNamespace(), gatewayProxy.GetName())
			matchingFields   = client.MatchingFields{indexer.IngressClassParametersRef: indexKey}
		)
		if err := r.List(ctx, &ingressClassList, matchingFields); err != nil {
			r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", indexKey)
			continue
		}
		for _, ingressClass := range ingressClassList.Items {
			requests = append(requests, r.listIngressForIngressClass(ctx, &ingressClass)...)
		}
	}

	// check if the secret is used by ApisixPluginConfig
	var pluginConfigList apiv2.ApisixPluginConfigList
	if err := r.List(ctx, &pluginConfigList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list plugin configs by secret", "secret", name)
	} else {
		// For each PluginConfig, find Ingresses that reference it
		for _, pc := range pluginConfigList.Items {
			var ingressList networkingv1.IngressList
			if err := r.List(ctx, &ingressList, client.MatchingFields{
				indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
			}); err != nil {
				r.Log.Error(err, "failed to list ingresses by plugin config", "pluginconfig", pc.GetName())
				continue
			}

			for _, ingress := range ingressList.Items {
				if MatchesIngressClass(r.Client, r.Log, &ingress) {
					requests = append(requests, reconcile.Request{
						NamespacedName: client.ObjectKey{
							Namespace: ingress.Namespace,
							Name:      ingress.Name,
						},
					})
				}
			}
		}
	}

	return distinctRequests(requests)
}

func (r *IngressReconciler) listIngressForBackendTrafficPolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	v, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}
	var namespacedNameMap = make(map[types.NamespacedName]struct{})
	ingresses := []networkingv1.Ingress{}
	for _, ref := range v.Spec.TargetRefs {
		service := &corev1.Service{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: v.Namespace,
			Name:      string(ref.Name),
		}, service); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get service", "namespace", v.Namespace, "name", ref.Name)
			}
			continue
		}
		ingressList := &networkingv1.IngressList{}
		if err := r.List(ctx, ingressList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(v.GetNamespace(), string(ref.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list HTTPRoutes for BackendTrafficPolicy", "namespace", v.GetNamespace(), "ref", ref.Name)
			return nil
		}
		ingresses = append(ingresses, ingressList.Items...)
	}
	for _, ins := range ingresses {
		key := types.NamespacedName{
			Namespace: ins.Namespace,
			Name:      ins.Name,
		}
		if _, ok := namespacedNameMap[key]; !ok {
			namespacedNameMap[key] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: key,
			})
		}
	}
	return requests
}

func (r *IngressReconciler) listIngressForGenericEvent(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	switch obj.(type) {
	case *v1alpha1.BackendTrafficPolicy:
		return r.listIngressForBackendTrafficPolicy(ctx, obj)
	case *v1alpha1.HTTPRoutePolicy:
		return r.listIngressesByHTTPRoutePolicy(ctx, obj)
	default:
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}
}

func (r *IngressReconciler) listIngressesByHTTPRoutePolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	httpRoutePolicy, ok := obj.(*v1alpha1.HTTPRoutePolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to HTTPRoutePolicy")
		return nil
	}

	var keys = make(map[types.NamespacedName]struct{})
	for _, ref := range httpRoutePolicy.Spec.TargetRefs {
		if ref.Kind != internaltypes.KindIngress {
			continue
		}
		key := types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      string(ref.Name),
		}
		if _, ok := keys[key]; ok {
			continue
		}

		var ingress networkingv1.Ingress
		if err := r.Get(ctx, key, &ingress); err != nil {
			r.Log.Error(err, "failed to get Ingress By HTTPRoutePolicy targetRef", "namespace", key.Namespace, "name", key.Name)
			continue
		}
		keys[key] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: key})
	}
	return
}

// processTLS process the TLS configuration of the ingress
func (r *IngressReconciler) processTLS(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		secret := corev1.Secret{}
		if err := r.Get(tctx, client.ObjectKey{
			Namespace: ingress.Namespace,
			Name:      tls.SecretName,
		}, &secret); err != nil {
			r.Log.Error(err, "failed to get secret", "namespace", ingress.Namespace, "name", tls.SecretName)
			return err
		}

		if secret.Data == nil {
			log.Warnw("secret data is nil", zap.String("secret", secret.Namespace+"/"+secret.Name))
			continue
		}

		// add the secret to the translate context
		tctx.Secrets[types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}] = &secret
	}

	return nil
}

// processBackends process the backend services of the ingress
func (r *IngressReconciler) processBackends(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	var terr error

	// process all the backend services in the rules
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			service := path.Backend.Service
			if err := r.processBackendService(tctx, ingress.Namespace, service); err != nil {
				terr = err
			}
		}
	}
	return terr
}

// processBackendService process a single backend service
func (r *IngressReconciler) processBackendService(tctx *provider.TranslateContext, namespace string, backendService *networkingv1.IngressServiceBackend) error {
	// get the service
	var service corev1.Service
	serviceNS := types.NamespacedName{
		Namespace: namespace,
		Name:      backendService.Name,
	}
	if err := r.Get(tctx, serviceNS, &service); err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.Log.Info("service not found", "namespace", namespace, "name", backendService.Name)
			return nil
		}
		return err
	}

	if service.Spec.Type == corev1.ServiceTypeExternalName {
		tctx.Services[serviceNS] = &service
		return nil
	}

	// verify if the port exists
	var portExists bool
	if backendService.Port.Number != 0 {
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Port == backendService.Port.Number {
				portExists = true
				break
			}
		}
	} else if backendService.Port.Name != "" {
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Name == backendService.Port.Name {
				portExists = true
				break
			}
		}
	}

	if !portExists {
		err := fmt.Errorf("port(name: %s, number: %d) not found in service %s/%s", backendService.Port.Name, backendService.Port.Number, namespace, backendService.Name)
		r.Log.Error(err, "service port not found")
		return err
	}

	// Collect endpoints with EndpointSlice support
	if err := resolveServiceEndpoints(tctx, r.Client, serviceNS, r.supportsEndpointSlice, nil); err != nil {
		r.Log.Error(err, "failed to collect endpoints", "namespace", namespace, "name", backendService.Name)
		return err
	}
	tctx.Services[serviceNS] = &service
	return nil
}

// processPluginConfig process the plugin config annotation of the ingress
func (r *IngressReconciler) processPluginConfig(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	pluginConfigName := ingress.Annotations[annotations.AnnotationsPluginConfigName]
	if pluginConfigName == "" {
		return nil
	}

	var (
		pc = apiv2.ApisixPluginConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginConfigName,
				Namespace: ingress.Namespace,
			},
		}
		pcNN = utils.NamespacedName(&pc)
	)

	if err := r.Get(tctx, pcNN, &pc); err != nil {
		r.Log.Error(err, "failed to get ApisixPluginConfig", "pluginconfig", pcNN)
		return err
	}

	// Check if ApisixPluginConfig has IngressClassName and if it matches
	if pc.Spec.IngressClassName != "" {
		ingressClassName := internaltypes.GetEffectiveIngressClassName(ingress)
		if ingressClassName != pc.Spec.IngressClassName {
			var pcIC networkingv1.IngressClass
			if err := r.Get(tctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &pcIC); err != nil {
				r.Log.Error(err, "failed to get IngressClass for ApisixPluginConfig", "ingressclass", pc.Spec.IngressClassName, "pluginconfig", pcNN)
				return nil
			}
			if !matchesController(pcIC.Spec.Controller) {
				r.Log.V(1).Info("ApisixPluginConfig references IngressClass with non-matching controller", "pluginconfig", pcNN, "ingressclass", pc.Spec.IngressClassName)
				return nil
			}
		}
	}

	tctx.ApisixPluginConfigs[pcNN] = &pc

	// Also check secrets referenced by plugin config
	for _, plugin := range pc.Spec.Plugins {
		if !plugin.Enable {
			continue
		}
		if plugin.SecretRef == "" {
			continue
		}
		var (
			secret = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      plugin.SecretRef,
					Namespace: ingress.Namespace,
				},
			}
			secretNN = utils.NamespacedName(&secret)
		)
		if err := r.Get(tctx, secretNN, &secret); err != nil {
			r.Log.Error(err, "failed to get Secret for ApisixPluginConfig", "secret", secretNN, "pluginconfig", pcNN)
			continue
		}
		tctx.Secrets[secretNN] = &secret
	}

	return nil
}

// updateStatus update the status of the ingress
func (r *IngressReconciler) updateStatus(ctx context.Context, tctx *provider.TranslateContext, ingress *networkingv1.Ingress, ingressClass *networkingv1.IngressClass) error {
	var loadBalancerStatus networkingv1.IngressLoadBalancerStatus

	ingressClassKind := utils.NamespacedNameKind(ingressClass)

	gatewayProxy, ok := tctx.GatewayProxies[ingressClassKind]
	if !ok {
		log.Debugw("no gateway proxy found for ingress class", zap.String("ingressClass", ingressClass.Name))
		return nil
	}

	// 1. use the IngressStatusAddress in the config
	statusAddresses := gatewayProxy.Spec.StatusAddress
	if len(statusAddresses) > 0 {
		for _, addr := range statusAddresses {
			if addr == "" {
				continue
			}
			loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
				IP: addr,
			})
		}
	} else {
		// 2. if the IngressStatusAddress is not configured, try to use the PublishService
		publishService := gatewayProxy.Spec.PublishService
		if publishService != "" {
			// parse the namespace/name format
			namespace, name, err := SplitMetaNamespaceKey(publishService)
			if err != nil {
				return fmt.Errorf("invalid ingress-publish-service format: %s, expected format: namespace/name", publishService)
			}
			// if the namespace is not specified, use the ingress namespace
			if namespace == "" {
				namespace = ingress.Namespace
			}

			svc := &corev1.Service{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc); err != nil {
				return fmt.Errorf("failed to get publish service %s: %w", publishService, err)
			}

			if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
				// get the LoadBalancer IP and Hostname of the service
				for _, ip := range svc.Status.LoadBalancer.Ingress {
					if ip.IP != "" {
						loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
							IP: ip.IP,
						})
					}
					if ip.Hostname != "" {
						loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, networkingv1.IngressLoadBalancerIngress{
							Hostname: ip.Hostname,
						})
					}
				}
			}
		}
	}

	// update the load balancer status
	if len(loadBalancerStatus.Ingress) > 0 && !reflect.DeepEqual(ingress.Status.LoadBalancer, loadBalancerStatus) {
		ingress.Status.LoadBalancer = loadBalancerStatus
		r.Updater.Update(status.Update{
			NamespacedName: utils.NamespacedName(ingress),
			Resource:       ingress.DeepCopy(),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*networkingv1.Ingress).DeepCopy()
				cp.Status = ingress.Status
				return cp
			}),
		})
		return nil
	}

	return nil
}

// listIngressesForGatewayProxy list all ingresses that use a specific gateway proxy
func (r *IngressReconciler) listIngressesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listIngressForIngressClass)
}

// listIngressesForPluginConfig list all ingresses that use a specific plugin config
func (r *IngressReconciler) listIngressesForPluginConfig(ctx context.Context, obj client.Object) []reconcile.Request {
	pc, ok := obj.(*apiv2.ApisixPluginConfig)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to ApisixPluginConfig")
		return nil
	}

	// First check if the ApisixPluginConfig has matching IngressClassName
	if pc.Spec.IngressClassName != "" {
		var ic networkingv1.IngressClass
		if err := r.Get(ctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &ic); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get IngressClass for ApisixPluginConfig", "pluginconfig", pc.Name)
			}
			return nil
		}
		if !matchesController(ic.Spec.Controller) {
			return nil
		}
	}

	var ingressList networkingv1.IngressList
	if err := r.List(ctx, &ingressList, client.MatchingFields{
		indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by plugin config", "pluginconfig", pc.Name)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress) {
			requests = append(requests, reconcile.Request{
				NamespacedName: utils.NamespacedName(&ingress),
			})
		}
	}
	return requests
}
