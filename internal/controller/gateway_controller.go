package controller

import (
	"context"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"
)

// -----------------------------------------------------------------------------
// Gateway Controller - GatewayReconciler
// -----------------------------------------------------------------------------

// GatewayReconciler reconciles a Gateway object.
type GatewayReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.Gateway{}).
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

	if gateway.Spec.GatewayClassName == "" {

	}

	gateway.Status.Conditions = append(gateway.Status.Conditions, meta.Condition{
		Type:   "Reconciled",
		Status: "True",
		Reason: "ReconciliationSucceeded",
	})

	if err := r.Status().Update(ctx, &gateway); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
