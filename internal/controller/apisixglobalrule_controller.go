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

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// ApisixGlobalRuleReconciler reconciles a ApisixGlobalRule object
type ApisixGlobalRuleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
}

// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules/status,verbs=get;update;patch

// Reconcile implements the reconciliation logic for ApisixGlobalRule
func (r *ApisixGlobalRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("apisixglobalrule", req.NamespacedName)
	log.Info("reconciling")

	var globalRule apiv2.ApisixGlobalRule
	if err := r.Get(ctx, req.NamespacedName, &globalRule); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("global rule not found, possibly deleted")
			// Create a minimal object for deletion
			globalRule.Namespace = req.Namespace
			globalRule.Name = req.Name
			globalRule.TypeMeta = metav1.TypeMeta{
				Kind:       "ApisixGlobalRule",
				APIVersion: apiv2.GroupVersion.String(),
			}
			// Delete from provider
			if err := r.Provider.Delete(ctx, &globalRule); err != nil {
				log.Error(err, "failed to delete global rule from provider")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get ApisixGlobalRule")
		return ctrl.Result{}, err
	}

	// Check if the global rule is being deleted
	if !globalRule.DeletionTimestamp.IsZero() {
		if err := r.Provider.Delete(ctx, &globalRule); err != nil {
			log.Error(err, "failed to delete global rule from provider")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Sync the global rule to APISIX
	if err := r.Provider.Update(ctx, &provider.TranslateContext{}, &globalRule); err != nil {
		log.Error(err, "failed to sync global rule to provider")
		// Update status with failure condition
		r.updateStatus(&globalRule, metav1.Condition{
			Type:               string(gatewayv1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: globalRule.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             "SyncFailed",
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	// Update status with success condition
	r.updateStatus(&globalRule, metav1.Condition{
		Type:               string(gatewayv1.RouteConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: globalRule.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             string(gatewayv1.RouteReasonAccepted),
		Message:            "The global rule has been accepted and synced to APISIX",
	})

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixGlobalRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixGlobalRule{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			),
		).
		Named("apisixglobalrule").
		Complete(r)
}

// checkIngressClass checks if the ApisixGlobalRule uses the ingress class that we control
func (r *ApisixGlobalRuleReconciler) checkIngressClass(obj client.Object) bool {
	globalRule, ok := obj.(*apiv2.ApisixGlobalRule)
	if !ok {
		return false
	}

	return r.matchesIngressClass(globalRule.Spec.IngressClassName)
}

// matchesIngressClass checks if the given ingress class name matches our controlled classes
func (r *ApisixGlobalRuleReconciler) matchesIngressClass(ingressClassName string) bool {
	if ingressClassName == "" {
		// Check for default ingress class
		ingressClassList := &networkingv1.IngressClassList{}
		if err := r.List(context.Background(), ingressClassList, client.MatchingFields{
			indexer.IngressClass: config.GetControllerName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingress classes")
			return false
		}

		// Find the ingress class that is marked as default
		for _, ic := range ingressClassList.Items {
			if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
				return true
			}
		}
		return false
	}

	// Check if the specified ingress class is controlled by us
	var ingressClass networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: ingressClassName}, &ingressClass); err != nil {
		r.Log.Error(err, "failed to get ingress class", "ingressClass", ingressClassName)
		return false
	}

	return matchesController(ingressClass.Spec.Controller)
}

// updateStatus updates the ApisixGlobalRule status with the given condition
func (r *ApisixGlobalRuleReconciler) updateStatus(globalRule *apiv2.ApisixGlobalRule, condition metav1.Condition) {
	r.Updater.Update(status.Update{
		NamespacedName: NamespacedName(globalRule),
		Resource:       globalRule.DeepCopy(),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			gr, ok := obj.(*apiv2.ApisixGlobalRule)
			if !ok {
				return nil
			}
			gr.Status.Conditions = []metav1.Condition{condition}
			return gr
		}),
	})
}
