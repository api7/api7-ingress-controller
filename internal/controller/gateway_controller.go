package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update

// GatewayReconciler reconciles a Gateway object.
type GatewayReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}).
		Watches(
			&gatewayv1.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayForGatewayClass),
			builder.WithPredicates(predicate.NewPredicateFuncs(r.matchesGatewayClass)),
		).
		Watches(
			&gatewayv1.HTTPRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForHTTPRoute),
		).
		Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := new(gatewayv1.Gateway)
	if err := r.Get(ctx, req.NamespacedName, gateway); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !r.checkGatewayClass(gateway) {
		return ctrl.Result{}, nil
	}

	conditionProgrammedStatus, conditionProgrammedMsg := true, "Programmed"

	cfg := config.GetFirstGatewayConfig()

	var addrs []gatewayv1.GatewayStatusAddress

	if len(gateway.Status.Addresses) != len(cfg.Addresses) {
		for _, addr := range cfg.Addresses {
			addrs = append(addrs,
				gatewayv1.GatewayStatusAddress{
					Value: addr,
				},
			)
		}
	}

	r.Log.Info("gateway has been accepted", "gateway", gateway.GetName())

	ListenerStatuses, err := getListenerStatus(ctx, r.Client, gateway)
	if err != nil {
		return ctrl.Result{}, err
	}

	accepted := SetGatewayConditionAccepted(gateway, true, acceptedMessage("gateway"))
	Programmed := SetGatewayConditionProgrammed(gateway, conditionProgrammedStatus, conditionProgrammedMsg)
	if accepted || Programmed || len(addrs) > 0 || len(ListenerStatuses) > 0 {
		if len(addrs) > 0 {
			gateway.Status.Addresses = addrs
		}
		if len(ListenerStatuses) > 0 {
			gateway.Status.Listeners = ListenerStatuses
		}

		return ctrl.Result{}, r.Status().Update(ctx, gateway)
	}
	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) matchesGatewayClass(obj client.Object) bool {
	gateway, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}
	return matchesController(string(gateway.Spec.ControllerName))
}

func (r *GatewayReconciler) matchesGatewayForControlPlaneConfig(obj client.Object) bool {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}
	cfg := config.GetControlPlaneConfigByGatewatName(gateway.GetName())
	ok = true
	if cfg == nil {
		ok = false
	}
	return ok
}

func (r *GatewayReconciler) listGatewayForGatewayClass(ctx context.Context, gatewayClass client.Object) []reconcile.Request {
	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(context.Background(), gatewayList); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway class",
			"gatewayclass", gatewayClass.GetName(),
		)
		return nil
	}

	/*
		gateways := []gatewayv1.Gateway{}
		for _, gateway := range gatewayList.Items {
			if cp := config.GetControlPlaneConfigByGatewatName(gateway.GetName()); cp != nil {
				gateways = append(gateways, gateway)
			}
		}
	*/
	return reconcileGatewaysMatchGatewayClass(gatewayClass, gatewayList.Items)
}

func (r *GatewayReconciler) checkGatewayClass(gateway *gatewayv1.Gateway) bool {
	gatewayClass := &gatewayv1.GatewayClass{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		r.Log.Error(err, "failed to get gateway class", "gatewayclass", gateway.Spec.GatewayClassName)
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}

func (r *GatewayReconciler) listGatewaysForHTTPRoute(_ context.Context, obj client.Object) []reconcile.Request {
	httpRoute, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		r.Log.Error(
			fmt.Errorf("unexpected object type"),
			"HTTPRoute watch predicate received unexpected object type",
			"expected", "*gatewayapi.HTTPRoute", "found", reflect.TypeOf(obj),
		)
		return nil
	}
	recs := []reconcile.Request{}
	for _, routeParentStatus := range httpRoute.Status.Parents {
		gatewayNamespace := httpRoute.GetNamespace()
		parentRef := routeParentStatus.ParentRef
		if parentRef.Group != nil && *parentRef.Group != gatewayv1.GroupName {
			continue
		}
		if parentRef.Kind != nil && *parentRef.Kind != "Gateway" {
			continue
		}
		if parentRef.Namespace != nil {
			gatewayNamespace = string(*parentRef.Namespace)
		}

		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gatewayNamespace,
				Name:      string(parentRef.Name),
			},
		})
	}

	fmt.Println("recs:", recs)
	return recs
}
