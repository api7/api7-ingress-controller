// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// ApisixConsumerReconciler reconciles a ApisixConsumer object
type ApisixConsumerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
	Updater  status.Updater
}

// Reconcile FIXME: implement the reconcile logic (For now, it dose nothing other than directly accepting)
func (r *ApisixConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("reconcile", "request", req.NamespacedName)

	var ac apiv2.ApisixConsumer
	if err := r.Get(ctx, req.NamespacedName, &ac); err != nil {
		r.Log.Error(err, "failed to get ApisixConsumer", "request", req.NamespacedName)
		return ctrl.Result{}, err
	}

	ac.Status.Conditions = []metav1.Condition{
		{
			Type:               string(gatewayv1.RouteConditionAccepted),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: ac.GetGeneration(),
			LastTransitionTime: metav1.Now(),
			Reason:             string(gatewayv1.RouteReasonAccepted),
		},
	}

	if err := r.Status().Update(ctx, &ac); err != nil {
		r.Log.Error(err, "failed to update status", "request", req.NamespacedName)
		return ctrl.Result{}, err
	}

	r.Updater.Update(status.Update{
		Resource: &apiv2.ApisixConsumer{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			acT, ok := obj.(*apiv2.ApisixConsumer)
			if !ok {
				err := fmt.Errorf("expected ApisixConsumer, got %T", obj)
				panic(err)
			}
			acCopy := acT.DeepCopy()
			acCopy.Status = acT.Status
			return acCopy
		}),
	})
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixConsumer{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			)).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixConsumerForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixConsumerForGatewayProxy),
		).
		Named("apisixconsumer").
		Complete(r)
}

func (r *ApisixConsumerReconciler) checkIngressClass(obj client.Object) bool {
	ac, ok := obj.(*apiv2.ApisixConsumer)
	if !ok {
		return false
	}

	return matchesIngressClass(r.Client, r.Log, ac.Spec.IngressClassName)
}

func (r *ApisixConsumerReconciler) listApisixConsumerForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listApisixConsumerForIngressClass)
}

func (r *ApisixConsumerReconciler) listApisixConsumerForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	return ListMatchingRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixConsumerList{},
		func(obj client.Object) bool {
			ac, ok := obj.(*apiv2.ApisixConsumer)
			if !ok {
				r.Log.Error(fmt.Errorf("expected ApisixConsumer, got %T", obj), "failed to match object type")
				return false
			}
			return (IsDefaultIngressClass(ingressClass) && ac.Spec.IngressClassName == "") || ac.Spec.IngressClassName == ingressClass.Name
		},
	)
}

func (r *ApisixConsumerReconciler) process() {
	return
}
