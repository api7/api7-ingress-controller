package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// HTTPRouteReconciler reconciles a GatewayClass object.
type HTTPRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger

	Provider provider.Provider
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		Watches(&v1alpha1.HTTPRoutePolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRouteByHTTPRoutePolicy),
		).
		Complete(r)
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

	tctx := provider.NewDefaultTranslateContext()

	if err := r.processHTTPRoute(tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	if err := r.processHTTPRoutePolicies(tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	if err := r.processHTTPRouteBackendRefs(tctx); err != nil {
		resolveRefStatus = status{
			status: false,
			msg:    err.Error(),
		}
	}

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

func (r *HTTPRouteReconciler) listHTTPRouteByHTTPRoutePolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	httpRoutePolicy, ok := obj.(*v1alpha1.HTTPRoutePolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to HTTPRoutePolicy")
		return nil
	}

	var keys = make(map[ancestorRefKey]struct{})
	for _, ref := range httpRoutePolicy.Spec.TargetRefs {
		if ref.Kind == "HTTPRoute" {
			key := ancestorRefKey{
				Group:     gatewayv1.GroupName,
				Kind:      "HTTPRoute",
				Namespace: gatewayv1.Namespace(obj.GetNamespace()),
				Name:      ref.Name,
			}
			if ref.SectionName != nil {
				key.SectionName = *ref.SectionName
			}
			keys[key] = struct{}{}
		}
	}
	for key := range keys {
		var httpRoute gatewayv1.HTTPRoute
		if err := r.Get(ctx, client.ObjectKey{Namespace: string(key.Namespace), Name: string(key.Name)}, &httpRoute); err != nil {
			r.Log.Error(err, "failed to get httproute by HTTPRoutePolicy targetRef", "namespace", obj.GetNamespace(), "name", obj.GetName())
			if err := r.updateHTTPRoutePolicyStatus(key, *httpRoutePolicy, false, string(v1alpha2.PolicyReasonTargetNotFound), "not found HTTPRoute"); err != nil {
				r.Log.Error(err, "failed to update HTTPRoutePolicy Status")
			}
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: string(key.Namespace),
				Name:      string(key.Name),
			},
		})
	}

	if err := r.Status().Update(ctx, httpRoutePolicy); err != nil {
		r.Log.Error(err, "failed to update HTTPRoutePolicy status", "namespace", obj.GetNamespace(), "name", obj.GetName())
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
		if err := r.Get(context.TODO(), client.ObjectKey{
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

		endpointSliceList := new(discoveryv1.EndpointSliceList)
		if err := r.List(context.TODO(), endpointSliceList,
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

func (r *HTTPRouteReconciler) processHTTPRoute(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	var terror error
	for _, rule := range httpRoute.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				pluginconfig := new(v1alpha1.PluginConfig)
				if err := r.Get(context.Background(), client.ObjectKey{
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
