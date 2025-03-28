package controller

import (
	"context"

	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConsumerReconciler  reconciles a Gateway object.
type ConsumerReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Consumer{}).
		Complete(r)
}

func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	consumer := new(v1alpha1.Consumer)
	if err := r.Get(ctx, req.NamespacedName, consumer); err != nil {
		if client.IgnoreNotFound(err) == nil {
			consumer.Namespace = req.Namespace
			consumer.Name = req.Name

			consumer.TypeMeta = metav1.TypeMeta{
				Kind:       "Consumer",
				APIVersion: v1alpha1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, consumer); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
