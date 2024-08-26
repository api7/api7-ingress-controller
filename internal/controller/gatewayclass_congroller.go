package controller

import (
	"context"
	"fmt"

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/go-logr/logr"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update

// GatewayClassReconciler reconciles a GatewayClass object.
type GatewayClassReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.GatewayClass{}).
		WithEventFilter(predicate.NewPredicateFuncs(r.GatewayClassFilter)).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var gc gatewayapi.GatewayClass
	if err := r.Get(ctx, req.NamespacedName, &gc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	condition := meta.Condition{
		Type:               string(gatewayapi.GatewayClassConditionStatusAccepted),
		Status:             meta.ConditionTrue,
		Reason:             string(gatewayapi.GatewayClassReasonAccepted),
		ObservedGeneration: gc.Generation,
		Message:            "the gatewayclass has been accepted by the api7-ingress-controller",
		LastTransitionTime: meta.Now(),
	}

	if !IsConditionPresentAndEqual(gc.Status.Conditions, condition) {
		r.Log.Info("gatewayclass has been accepted", "gatewayclass", gc.Name)
		setGatewayClassCondition(&gc, condition)
		if err := r.Status().Update(ctx, &gc); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *GatewayClassReconciler) GatewayClassFilter(obj client.Object) bool {
	gatewayClass, ok := obj.(*gatewayapi.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayClass")
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}

func matchesController(controllerName string) bool {
	return controllerName == config.ControllerConfig.ControllerName
}

func setGatewayClassCondition(gwc *gatewayapi.GatewayClass, newCondition meta.Condition) {
	newConditions := []meta.Condition{}
	for _, condition := range gwc.Status.Conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	gwc.Status.Conditions = newConditions
}
