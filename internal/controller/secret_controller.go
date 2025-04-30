package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

type SecretReconciler struct {
	client.Client

	Log     logr.Logger
	Indexer cache.Indexer

	GenericEvent chan event.GenericEvent
}

func (r *SecretReconciler) SetupWithManager(mgr manager.Manager) error {
	r.GenericEvent = GatewaySecretChan
	return ctrl.NewControllerManagedBy(mgr).
		Named("CoreV1Secret").
		WithOptions(controller.Options{
			CacheSyncTimeout: time.Second,
			LogConstructor: func(_ *reconcile.Request) logr.Logger {
				return r.Log
			},
		}).
		For(&corev1.Secret{}).
		Complete(r)
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.GenericEvent <- event.GenericEvent{
		Object: &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			},
		},
	}
	return ctrl.Result{}, nil
}
