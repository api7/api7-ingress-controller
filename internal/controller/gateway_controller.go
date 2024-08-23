package controller

import (
	"context"
	"fmt"

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/go-logr/logr"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"
)

func acceptedMessage(kind string) string {
	return fmt.Sprintf("the %s has been accepted by the api7-ingress-controller", kind)
}

// -----------------------------------------------------------------------------
// Gateway Controller - GatewayReconciler
// -----------------------------------------------------------------------------

// GatewayReconciler reconciles a Gateway object.
type GatewayReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.Gateway{},
			builder.WithPredicates(
				predicate.And(
					predicate.NewPredicateFuncs(r.matchesGatewayForControlPlaneConfig),
					predicate.NewPredicateFuncs(r.checkGatewayClass),
				),
			),
		).
		Watches(
			&gatewayapi.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayForGatewayClass),
			builder.WithPredicates(predicate.NewPredicateFuncs(r.matchesGatewayClass)),
		).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Complete(r)
}

// -----------------------------------------------------------------------------
// Gateway Controller - Reconciliation
// -----------------------------------------------------------------------------

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var gateway gatewayapi.Gateway
	if err := r.Get(ctx, req.NamespacedName, &gateway); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	condition := meta.Condition{
		Type:               string(gatewayapi.GatewayConditionAccepted),
		Status:             meta.ConditionTrue,
		Reason:             string(gatewayapi.GatewayReasonAccepted),
		ObservedGeneration: gateway.Generation,
		Message:            acceptedMessage("gateway"),
		LastTransitionTime: meta.Now(),
	}
	if !IsConditionPresentAndEqual(gateway.Status.Conditions, condition) {
		r.Log.Info("gateway has been accepted", "gateway", gateway.Name)
		setGatewayCondition(&gateway, condition)
		if err := r.Status().Update(ctx, &gateway); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) matchesGatewayClass(obj client.Object) bool {
	gateway, ok := obj.(*gatewayapi.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}
	return matchesController(string(gateway.Spec.ControllerName))
}

func (r *GatewayReconciler) matchesGatewayForControlPlaneConfig(obj client.Object) bool {
	gateway, ok := obj.(*gatewayapi.Gateway)
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
	gatewayList := &gatewayapi.GatewayList{}
	if err := r.List(context.Background(), gatewayList); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway class",
			"gatewayclass", gatewayClass.GetName(),
		)
		return nil
	}

	gateways := []gatewayapi.Gateway{}
	for _, gateway := range gatewayList.Items {
		if cp := config.GetControlPlaneConfigByGatewatName(gateway.GetName()); cp != nil {
			gateways = append(gateways, gateway)
		}
	}
	return reconcileGatewaysMatchGatewayClass(gatewayClass, gateways)
}

func (r *GatewayReconciler) checkGatewayClass(obj client.Object) bool {
	gateway, ok := obj.(*gatewayapi.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}

	gatewayClass := &gatewayapi.GatewayClass{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		r.Log.Error(err, "failed to get gateway class", "gatewayclass", gateway.Spec.GatewayClassName)
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}
