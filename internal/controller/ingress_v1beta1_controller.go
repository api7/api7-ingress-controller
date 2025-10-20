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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
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
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	controllerutils "github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// IngressV1beta1Reconciler reconciles a networking.k8s.io/v1beta1 Ingress object.
type IngressV1beta1Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider     provider.Provider
	genericEvent chan event.GenericEvent

	Updater status.Updater
	Readier readiness.ReadinessManager

	supportsEndpointSlice bool

	v1Helper *IngressReconciler
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressV1beta1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.genericEvent = make(chan event.GenericEvent, 100)

	r.supportsEndpointSlice = pkgutils.HasAPIResource(mgr, &discoveryv1.EndpointSlice{})

	r.v1Helper = &IngressReconciler{
		Client:                r.Client,
		Scheme:                r.Scheme,
		Log:                   r.Log,
		Provider:              r.Provider,
		genericEvent:          r.genericEvent,
		Updater:               r.Updater,
		Readier:               r.Readier,
		supportsEndpointSlice: r.supportsEndpointSlice,
	}

	eventFilters := []predicate.Predicate{
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
		predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
	}

	if !r.supportsEndpointSlice {
		eventFilters = append(eventFilters, predicate.NewPredicateFuncs(TypePredicate[*corev1.Endpoints]()))
	}

	apiVersion := networkingv1beta1.SchemeGroupVersion.String()

	bdr := ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1beta1.Ingress{},
			builder.WithPredicates(
				MatchesIngressClassPredicate(r.Client, r.Log, apiVersion),
			),
		).
		WithEventFilter(predicate.Or(eventFilters...))

	if pkgutils.HasAPIResource(mgr, &networkingv1.IngressClass{}) {
		bdr = bdr.Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressForIngressClassV1),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressControllerV1),
			),
		)
	} else {
		bdr = bdr.Watches(
			&networkingv1beta1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressForIngressClassV1beta1),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressControllerV1beta1),
			),
		)
	}

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
		WatchesRawSource(
			source.Channel(
				r.genericEvent,
				handler.EnqueueRequestsFromMapFunc(r.listIngressForGenericEvent),
			),
		).
		Complete(r)
}

func (r *IngressV1beta1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&networkingv1beta1.Ingress{}, req.NamespacedName)

	ingress := new(networkingv1beta1.Ingress)
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		if client.IgnoreNotFound(err) == nil {
			if err := r.v1Helper.updateHTTPRoutePolicyStatusOnDeleting(ctx, req.NamespacedName); err != nil {
				return ctrl.Result{}, err
			}
			ingress.Namespace = req.Namespace
			ingress.Name = req.Name

			converted := pkgutils.ConvertIngressV1beta1ToV1(ingress)
			converted.TypeMeta = metav1.TypeMeta{
				Kind:       KindIngress,
				APIVersion: networkingv1.SchemeGroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, converted); err != nil {
				r.Log.Error(err, "failed to delete ingress resources", "ingress", ingress.Name)
				return ctrl.Result{}, err
			}
			r.Log.Info("deleted ingress resources", "ingress", ingress.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reconciling ingress v1beta1", "ingress", ingress.Name)

	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := FindMatchingIngressClassByObject(tctx, r.Client, r.Log, ingress, networkingv1beta1.SchemeGroupVersion.String())
	if err != nil {
		if err := r.Provider.Delete(ctx, pkgutils.ConvertIngressV1beta1ToV1(ingress)); err != nil {
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

	convertedIngress := pkgutils.ConvertIngressV1beta1ToV1(ingress)
	if err := ProcessIngressClassParameters(tctx, r.Client, r.Log, convertedIngress, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, err
	}

	if err := r.v1Helper.processTLS(tctx, convertedIngress); err != nil {
		r.Log.Error(err, "failed to process TLS configuration", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	if err := r.v1Helper.processBackends(tctx, convertedIngress); err != nil {
		r.Log.Error(err, "failed to process backend services", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	if err := r.v1Helper.processHTTPRoutePolicies(tctx, convertedIngress); err != nil {
		r.Log.Error(err, "failed to process HTTPRoutePolicy", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)

	if err := r.Provider.Update(ctx, tctx, convertedIngress); err != nil {
		r.Log.Error(err, "failed to update ingress resources", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	UpdateStatus(r.Updater, r.Log, tctx)

	if err := r.updateStatus(ctx, tctx, ingress, ingressClass); err != nil {
		r.Log.Error(err, "failed to update ingress status", "ingress", ingress.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IngressV1beta1Reconciler) matchesIngressControllerV1(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *IngressV1beta1Reconciler) matchesIngressControllerV1beta1(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1beta1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass v1beta1")
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *IngressV1beta1Reconciler) listIngressForIngressClassV1(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return nil
	}

	ingressList := &networkingv1beta1.IngressList{}
	if IsDefaultIngressClass(ingressClass) {
		if err := r.List(ctx, ingressList); err != nil {
			r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
			return nil
		}
		return r.requestsForIngressList(ingressList.Items, ingressClass.GetName())
	}

	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.IngressClassRef: ingressClass.GetName(),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses for ingress class", "ingressclass", ingressClass.GetName())
		return nil
	}
	return r.requestsForIngressList(ingressList.Items, "")
}

func (r *IngressV1beta1Reconciler) listIngressForIngressClassV1beta1(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1beta1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass v1beta1")
		return nil
	}
	converted := pkgutils.ConvertToIngressClassV1(ingressClass)
	return r.listIngressForIngressClassV1(ctx, converted)
}

func (r *IngressV1beta1Reconciler) requestsForIngressList(items []networkingv1beta1.Ingress, ingressClassName string) []reconcile.Request {
	requests := make([]reconcile.Request, 0, len(items))
	for _, ingress := range items {
		if ingressClassName != "" {
			class := ptr.Deref(ingress.Spec.IngressClassName, "")
			if class != "" && class != ingressClassName {
				continue
			}
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: ingress.Namespace,
				Name:      ingress.Name,
			},
		})
	}
	return requests
}

func (r *IngressV1beta1Reconciler) listIngressesByService(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}

	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	ingressList := &networkingv1beta1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by service", "service", serviceName)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, networkingv1beta1.SchemeGroupVersion.String()) {
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

func (r *IngressV1beta1Reconciler) listIngressesByEndpoints(ctx context.Context, obj client.Object) []reconcile.Request {
	endpoint, ok := obj.(*corev1.Endpoints)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Endpoints")
		return nil
	}

	namespace := endpoint.GetNamespace()
	serviceName := endpoint.GetName()

	ingressList := &networkingv1beta1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by service", "service", serviceName)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, networkingv1beta1.SchemeGroupVersion.String()) {
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

func (r *IngressV1beta1Reconciler) listIngressesBySecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Secret")
		return nil
	}

	namespace := secret.GetNamespace()
	name := secret.GetName()

	ingressList := &networkingv1beta1.IngressList{}
	if err := r.List(ctx, ingressList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list ingresses by secret", "secret", name)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(ingressList.Items))
	for _, ingress := range ingressList.Items {
		if MatchesIngressClass(r.Client, r.Log, &ingress, networkingv1beta1.SchemeGroupVersion.String()) {
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
			ingressClassList networkingv1beta1.IngressClassList
			indexKey         = indexer.GenIndexKey(gatewayProxy.GetNamespace(), gatewayProxy.GetName())
			matchingFields   = client.MatchingFields{indexer.IngressClassParametersRef: indexKey}
		)
		if err := r.List(ctx, &ingressClassList, matchingFields); err != nil {
			r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", indexKey)
			continue
		}
		for _, ingressClass := range ingressClassList.Items {
			requests = append(requests, r.listIngressForIngressClassV1beta1(ctx, &ingressClass)...)
		}
	}

	return distinctRequests(requests)
}

func (r *IngressV1beta1Reconciler) listIngressForBackendTrafficPolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	v, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}
	var namespacedNameMap = make(map[types.NamespacedName]struct{})
	ingresses := []networkingv1beta1.Ingress{}
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
		ingressList := &networkingv1beta1.IngressList{}
		if err := r.List(ctx, ingressList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(v.GetNamespace(), string(ref.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list Ingresses for BackendTrafficPolicy", "namespace", v.GetNamespace(), "ref", ref.Name)
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

func (r *IngressV1beta1Reconciler) listIngressForGenericEvent(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
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

func (r *IngressV1beta1Reconciler) listIngressesByHTTPRoutePolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
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

		var ingress networkingv1beta1.Ingress
		if err := r.Get(ctx, key, &ingress); err != nil {
			r.Log.Error(err, "failed to get Ingress(v1beta1) By HTTPRoutePolicy targetRef", "namespace", key.Namespace, "name", key.Name)
			continue
		}
		keys[key] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: key})
	}
	return
}

func (r *IngressV1beta1Reconciler) listIngressesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listIngressForIngressClassV1beta1)
}

func (r *IngressV1beta1Reconciler) updateStatus(ctx context.Context, tctx *provider.TranslateContext, ingress *networkingv1beta1.Ingress, ingressClass *networkingv1.IngressClass) error {
	var loadBalancerStatus networkingv1.IngressLoadBalancerStatus

	ingressClassKind := controllerutils.NamespacedNameKind(ingressClass)

	gatewayProxy, ok := tctx.GatewayProxies[ingressClassKind]
	if !ok {
		log.Debugw("no gateway proxy found for ingress class", zap.String("ingressClass", ingressClass.Name))
		return nil
	}

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
		publishService := gatewayProxy.Spec.PublishService
		if publishService != "" {
			namespace, name, err := SplitMetaNamespaceKey(publishService)
			if err != nil {
				return fmt.Errorf("invalid ingress-publish-service format: %s, expected format: namespace/name", publishService)
			}
			if namespace == "" {
				namespace = ingress.Namespace
			}

			svc := &corev1.Service{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc); err != nil {
				return fmt.Errorf("failed to get publish service %s: %w", publishService, err)
			}

			if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
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

	currentStatus := pkgutils.ConvertIngressStatusV1beta1ToV1(ingress.Status)

	if len(loadBalancerStatus.Ingress) > 0 && !reflect.DeepEqual(currentStatus.LoadBalancer, loadBalancerStatus) {
		ingress.Status = pkgutils.ConvertIngressStatusV1ToV1beta1(networkingv1.IngressStatus{LoadBalancer: loadBalancerStatus})
		r.Updater.Update(status.Update{
			NamespacedName: controllerutils.NamespacedName(ingress),
			Resource:       ingress.DeepCopy(),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*networkingv1beta1.Ingress).DeepCopy()
				cp.Status = ingress.Status
				return cp
			}),
		})
		return nil
	}
	return nil
}
