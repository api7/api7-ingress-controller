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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
)

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
		For(&corev1.Secret{}). //builder.WithPredicates(r.predicateFuncs()),

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

func (r *SecretReconciler) predicateFuncs() predicate.Funcs {
	predicateFuncs := predicate.NewPredicateFuncs(func(object client.Object) bool {
		if _, ok := object.(*corev1.Secret); !ok {
			return false
		}
		key := indexer.GenIndexKey(object.GetNamespace(), object.GetName())
		refs, err := r.Indexer.ByIndex("referent", key)
		if err != nil {
			r.Log.Error(err, "failed to check whether secret referred", "namespace", object.GetNamespace(), "name", object.GetName())
			return false
		}
		return len(refs) > 0
	})
	predicateFuncs.DeleteFunc = func(_ event.DeleteEvent) bool {
		return true
	}
	return predicateFuncs
}
