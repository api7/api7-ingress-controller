package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

// HTTPRouteReconciler reconciles a GatewayClass object.
type HTTPRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger

	Provider provider.Provider

	genericEvent chan event.GenericEvent
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.genericEvent = make(chan event.GenericEvent, 100)

	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.HTTPRoute{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesByServiceBef),
		).
		Watches(&v1alpha1.PluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesByExtensionRef),
		).
		Watches(&gatewayv1.Gateway{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForGateway),
			builder.WithPredicates(
				predicate.Funcs{
					GenericFunc: func(e event.GenericEvent) bool {
						return false
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return false
					},
					CreateFunc: func(e event.CreateEvent) bool {
						return true
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						return true
					},
				},
			),
		).
		Watches(&v1alpha1.BackendTrafficPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForBackendTrafficPolicy),
			builder.WithPredicates(
				predicate.Funcs{
					GenericFunc: func(e event.GenericEvent) bool {
						return false
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return true
					},
					CreateFunc: func(e event.CreateEvent) bool {
						return true
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						oldObj, ok := e.ObjectOld.(*v1alpha1.BackendTrafficPolicy)
						newObj, ok2 := e.ObjectNew.(*v1alpha1.BackendTrafficPolicy)
						if !ok || !ok2 {
							return false
						}
						oldRefs := oldObj.Spec.TargetRefs
						newRefs := newObj.Spec.TargetRefs

						// 将旧引用转换为 Map
						oldRefMap := make(map[string]v1alpha1.BackendPolicyTargetReferenceWithSectionName)
						for _, ref := range oldRefs {
							key := fmt.Sprintf("%s/%s/%s", ref.Group, ref.Kind, ref.Name)
							oldRefMap[key] = ref
						}

						for _, ref := range newRefs {
							key := fmt.Sprintf("%s/%s/%s", ref.Group, ref.Kind, ref.Name)
							delete(oldRefMap, key)
						}
						if len(oldRefMap) > 0 {
							targetRefs := make([]v1alpha1.BackendPolicyTargetReferenceWithSectionName, 0, len(oldRefs))
							for _, ref := range oldRefMap {
								targetRefs = append(targetRefs, ref)
							}
							dump := oldObj.DeepCopy()
							dump.Spec.TargetRefs = targetRefs
							r.genericEvent <- event.GenericEvent{
								Object: dump,
							}
						}
						return true
					},
				},
			),
		).
		WatchesRawSource(
			source.Channel(
				r.genericEvent,
				handler.EnqueueRequestsFromMapFunc(r.listHTTPRouteForGenericEvent),
			),
		).
		Complete(r)
}

func (r *HTTPRouteReconciler) listHTTPRouteForGenericEvent(ctx context.Context, obj client.Object) []reconcile.Request {
	var namespacedNameMap = make(map[types.NamespacedName]struct{})
	requests := []reconcile.Request{}
	switch v := obj.(type) {
	case *v1alpha1.BackendTrafficPolicy:
		httprouteAll := []gatewayv1.HTTPRoute{}
		for _, ref := range v.Spec.TargetRefs {
			httprouteList := &gatewayv1.HTTPRouteList{}
			if err := r.List(ctx, httprouteList, client.MatchingFields{
				indexer.ServiceIndexRef: indexer.GenIndexKey(v.GetNamespace(), string(ref.Name)),
			}); err != nil {
				r.Log.Error(err, "failed to list HTTPRoutes for BackendTrafficPolicy", "namespace", v.GetNamespace(), "ref", ref.Name)
				return nil
			}
			httprouteAll = append(httprouteAll, httprouteList.Items...)
		}
		for _, hr := range httprouteAll {
			key := types.NamespacedName{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			}
			if _, ok := namespacedNameMap[key]; !ok {
				namespacedNameMap[key] = struct{}{}
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: hr.Namespace,
						Name:      hr.Name,
					},
				})
			}
		}
	default:
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
	}
	return requests
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	hr := new(gatewayv1.HTTPRoute)
	if err := r.Get(ctx, req.NamespacedName, hr); err != nil {
		if client.IgnoreNotFound(err) == nil {
			hr.Namespace = req.Namespace
			hr.Name = req.Name

			hr.TypeMeta = metav1.TypeMeta{
				Kind:       KindHTTPRoute,
				APIVersion: gatewayv1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, hr); err != nil {
				r.Log.Error(err, "failed to delete httproute", "httproute", hr)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	type status struct {
		status bool
		msg    string
	}

	resolveRefStatus := status{
		status: true,
		msg:    "backendRefs are resolved",
	}
	acceptStatus := status{
		status: true,
		msg:    "Route is accepted",
	}

	gateways, err := ParseRouteParentRefs(ctx, r.Client, hr, hr.Spec.ParentRefs)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(gateways) == 0 {
		return ctrl.Result{}, nil
	}

	tctx := provider.NewDefaultTranslateContext(ctx)

	tctx.RouteParentRefs = hr.Spec.ParentRefs
	rk := provider.ResourceKind{
		Kind:      hr.Kind,
		Namespace: hr.Namespace,
		Name:      hr.Name,
	}
	for _, gateway := range gateways {
		if err := ProcessGatewayProxy(r.Client, tctx, gateway.Gateway, rk); err != nil {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
	}

	if err := r.processHTTPRoute(tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	if err := r.processHTTPRouteBackendRefs(tctx); err != nil {
		resolveRefStatus = status{
			status: false,
			msg:    err.Error(),
		}
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)

	if err := r.Provider.Update(ctx, tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	// TODO: diff the old and new status
	hr.Status.Parents = make([]gatewayv1.RouteParentStatus, 0, len(gateways))
	for _, gateway := range gateways {
		parentStatus := gatewayv1.RouteParentStatus{}
		SetRouteParentRef(&parentStatus, gateway.Gateway.Name, gateway.Gateway.Namespace)
		for _, condition := range gateway.Conditions {
			parentStatus.Conditions = MergeCondition(parentStatus.Conditions, condition)
		}
		if gateway.ListenerName == "" {
			continue
		}
		SetRouteConditionAccepted(&parentStatus, hr.GetGeneration(), acceptStatus.status, acceptStatus.msg)
		SetRouteConditionResolvedRefs(&parentStatus, hr.GetGeneration(), resolveRefStatus.status, resolveRefStatus.msg)
		hr.Status.Parents = append(hr.Status.Parents, parentStatus)
	}
	if err := r.Status().Update(ctx, hr); err != nil {
		return ctrl.Result{}, err
	}
	UpdateStatus(r.Client, r.Log, tctx)
	return ctrl.Result{}, nil
}

func (r *HTTPRouteReconciler) listHTTPRoutesByServiceBef(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesByExtensionRef(ctx context.Context, obj client.Object) []reconcile.Request {
	pluginconfig, ok := obj.(*v1alpha1.PluginConfig)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := pluginconfig.GetNamespace()
	name := pluginconfig.GetName()

	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ExtensionRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by extension reference", "extension", name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesForBackendTrafficPolicy(ctx context.Context, obj client.Object) []reconcile.Request {
	policy, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}

	httprouteList := []gatewayv1.HTTPRoute{}
	for _, targetRef := range policy.Spec.TargetRefs {
		service := &corev1.Service{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: policy.Namespace,
			Name:      string(targetRef.Name),
		}, service); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get service", "namespace", policy.Namespace, "name", targetRef.Name)
			}
			continue
		}
		hrList := &gatewayv1.HTTPRouteList{}
		if err := r.List(ctx, hrList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(policy.Namespace, string(targetRef.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list httproutes by service reference", "service", targetRef.Name)
			return nil
		}
		httprouteList = append(httprouteList, hrList.Items...)
	}
	var namespacedNameMap = make(map[types.NamespacedName]struct{})
	requests := make([]reconcile.Request, 0, len(httprouteList))
	for _, hr := range httprouteList {
		key := types.NamespacedName{
			Namespace: hr.Namespace,
			Name:      hr.Name,
		}
		if _, ok := namespacedNameMap[key]; !ok {
			namespacedNameMap[key] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: hr.Namespace,
					Name:      hr.Name,
				},
			})
		}
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesForGateway(ctx context.Context, obj client.Object) []reconcile.Request {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
	}
	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by gateway", "gateway", gateway.Name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) processHTTPRouteBackendRefs(tctx *provider.TranslateContext) error {
	var terr error
	for _, backend := range tctx.BackendRefs {
		namespace := string(*backend.Namespace)
		name := string(backend.Name)

		if backend.Kind != nil && *backend.Kind != "Service" {
			terr = fmt.Errorf("kind %s is not supported", *backend.Kind)
			continue
		}

		if backend.Port == nil {
			terr = fmt.Errorf("port is required")
			continue
		}

		var service corev1.Service
		if err := r.Get(tctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &service); err != nil {
			terr = err
			continue
		}

		portExists := false
		for _, port := range service.Spec.Ports {
			if port.Port == int32(*backend.Port) {
				portExists = true
				break
			}
		}
		if !portExists {
			terr = fmt.Errorf("port %d not found in service %s", *backend.Port, name)
			continue
		}
		tctx.Services[client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}] = &service

		endpointSliceList := new(discoveryv1.EndpointSliceList)
		if err := r.List(tctx, endpointSliceList,
			client.InNamespace(namespace),
			client.MatchingLabels{
				discoveryv1.LabelServiceName: name,
			},
		); err != nil {
			r.Log.Error(err, "failed to list endpoint slices", "namespace", namespace, "name", name)
			terr = err
			continue
		}

		tctx.EndpointSlices[client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}] = endpointSliceList.Items

	}
	return terr
}

func (t *HTTPRouteReconciler) processHTTPRoute(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	var terror error
	for _, rule := range httpRoute.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				pluginconfig := new(v1alpha1.PluginConfig)
				if err := t.Get(context.Background(), client.ObjectKey{
					Namespace: httpRoute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}, pluginconfig); err != nil {
					terror = err
					continue
				}
				tctx.PluginConfigs[types.NamespacedName{
					Namespace: httpRoute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}] = pluginconfig
			}
		}
		for _, backend := range rule.BackendRefs {
			var kind string
			if backend.Kind == nil {
				kind = "service"
			} else {
				kind = strings.ToLower(string(*backend.Kind))
			}
			if kind != "service" {
				terror = fmt.Errorf("kind %s is not supported", kind)
				continue
			}

			var ns string
			if backend.Namespace == nil {
				ns = httpRoute.Namespace
			} else {
				ns = string(*backend.Namespace)
			}

			backendNs := gatewayv1.Namespace(ns)
			tctx.BackendRefs = append(tctx.BackendRefs, gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      backend.Name,
					Namespace: &backendNs,
					Port:      backend.Port,
				},
			})
		}
	}

	return terror
}
