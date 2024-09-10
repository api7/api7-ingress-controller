package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/controlplane"
	"github.com/api7/api7-ingress-controller/internal/controlplane/translator"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update

// HTTPRouteReconciler reconciles a GatewayClass object.
type HTTPRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log                logr.Logger
	ControlPalneClient controlplane.Controlplane
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.TODO(),
		&gatewayv1.HTTPRoute{},
		indexer.ServiceIndexRef,
		indexer.HTTPRouteServiceIndexFunc,
	); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.HTTPRoute{}).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesByServiceBef),
		).
		Complete(r)
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	hr := new(gatewayv1.HTTPRoute)
	if err := r.Get(ctx, req.NamespacedName, hr); err != nil {
		if client.IgnoreNotFound(err) == nil {
			hr.Namespace = req.Namespace
			hr.Name = req.Name

			if err := r.ControlPalneClient.Delete(ctx, hr); err != nil {
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

	tctx := &translator.TranslateContext{
		EndpointSlices: make(map[client.ObjectKey][]discoveryv1.EndpointSlice),
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

	if err := r.ControlPalneClient.Update(ctx, tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	// process the HTTPRoute

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

func (r *HTTPRouteReconciler) processHTTPRouteBackendRefs(tctx *translator.TranslateContext) error {
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

func (t *HTTPRouteReconciler) processHTTPRoute(tctx *translator.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	var terror error
	for _, rule := range httpRoute.Spec.Rules {
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
