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

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixPluginConfigReconciler reconciles a ApisixPluginConfig object
type ApisixPluginConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixPluginConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixPluginConfig{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
			),
		).
		Watches(&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixPluginConfigForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixPluginConfigForGatewayProxy),
		).
		Named("apisixpluginconfig").
		Complete(r)
}

func (r *ApisixPluginConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pc apiv2.ApisixPluginConfig
	if err := r.Get(ctx, req.NamespacedName, &pc); err != nil {
		if client.IgnoreNotFound(err) == nil {
			pc.Namespace = req.Namespace
			pc.Name = req.Name
			pc.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixPluginConfig,
				APIVersion: apiv2.GroupVersion.String(),
			}

			// if err := r.Provider.Delete(ctx, &pc); err != nil {
			// 	r.Log.Error(err, "failed to delete apisixpluginconfig", "apisixpluginconfig", pc)
			// 	return ctrl.Result{}, err
			// }
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var (
		tctx = provider.NewDefaultTranslateContext(ctx)
		ic   *networkingv1.IngressClass
		err  error
	)
	defer func() {
		r.updateStatus(&pc, err)
	}()

	if ic, err = r.getIngressClass(&pc); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processIngressClassParameters(ctx, tctx, &pc, ic); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processApisixPluginConfig(ctx, tctx, &pc); err != nil {
		return ctrl.Result{}, err
	}
	// if err = r.Provider.Update(ctx, tctx, &pc); err != nil {
	// 	err = ReasonError{
	// 		Reason:  string(apiv2.ConditionReasonSyncFailed),
	// 		Message: err.Error(),
	// 	}
	// 	r.Log.Error(err, "failed to process", "apisixpluginconfig", pc)
	// 	return ctrl.Result{}, err
	// }

	return ctrl.Result{}, nil
}

func (r *ApisixPluginConfigReconciler) processApisixPluginConfig(_ context.Context, _ *provider.TranslateContext, in *apiv2.ApisixPluginConfig) error {
	// Validate plugins
	for _, plugin := range in.Spec.Plugins {
		if plugin.Name == "" {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: "plugin name cannot be empty",
			}
		}
	}

	return nil
}

func (r *ApisixPluginConfigReconciler) listApisixPluginConfigForIngressClass(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	ic, ok := object.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	isDefaultIngressClass := IsDefaultIngressClass(ic)
	var pcList apiv2.ApisixPluginConfigList
	if err := r.List(ctx, &pcList); err != nil {
		return nil
	}
	for _, pc := range pcList.Items {
		if pc.Spec.IngressClassName == ic.Name || (isDefaultIngressClass && pc.Spec.IngressClassName == "") {
			requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&pc)})
		}
	}
	return requests
}

func (r *ApisixPluginConfigReconciler) listApisixPluginConfigForGatewayProxy(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	gp, ok := object.(*v1alpha1.GatewayProxy)
	if !ok {
		return nil
	}

	var icList networkingv1.IngressClassList
	if err := r.List(ctx, &icList); err != nil {
		r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", gp.GetName())
		return nil
	}

	for _, ic := range icList.Items {
		requests = append(requests, r.listApisixPluginConfigForIngressClass(ctx, &ic)...)
	}

	return requests
}

func (r *ApisixPluginConfigReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *ApisixPluginConfigReconciler) getIngressClass(pc *apiv2.ApisixPluginConfig) (*networkingv1.IngressClass, error) {
	if pc.Spec.IngressClassName == "" {
		return r.getDefaultIngressClass()
	}

	var ic networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: pc.Spec.IngressClassName}, &ic); err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *ApisixPluginConfigReconciler) getDefaultIngressClass() (*networkingv1.IngressClass, error) {
	var icList networkingv1.IngressClassList
	if err := r.List(context.Background(), &icList); err != nil {
		r.Log.Error(err, "failed to list ingress classes")
		return nil, err
	}
	for _, ic := range icList.Items {
		if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
			return &ic, nil
		}
	}
	return nil, ReasonError{
		Reason:  string(metav1.StatusReasonNotFound),
		Message: "default ingress class not found or does not match the controller",
	}
}

// processIngressClassParameters processes the IngressClass parameters that reference GatewayProxy
func (r *ApisixPluginConfigReconciler) processIngressClassParameters(ctx context.Context, tc *provider.TranslateContext, pc *apiv2.ApisixPluginConfig, ingressClass *networkingv1.IngressClass) error {
	if ingressClass == nil || ingressClass.Spec.Parameters == nil {
		return nil
	}

	var (
		ingressClassKind = utils.NamespacedNameKind(ingressClass)
		pcKind           = utils.NamespacedNameKind(pc)
		parameters       = ingressClass.Spec.Parameters
	)
	if parameters.APIGroup == nil || *parameters.APIGroup != v1alpha1.GroupVersion.Group || parameters.Kind != KindGatewayProxy {
		return nil
	}

	// check if the parameters reference GatewayProxy
	var (
		gatewayProxy v1alpha1.GatewayProxy
		ns           = parameters.Namespace
	)
	if ns == nil {
		ns = &pc.Namespace
	}

	if err := r.Get(ctx, client.ObjectKey{Namespace: *ns, Name: parameters.Name}, &gatewayProxy); err != nil {
		r.Log.Error(err, "failed to get GatewayProxy", "namespace", *ns, "name", parameters.Name)
		return err
	}

	tc.GatewayProxies[ingressClassKind] = gatewayProxy
	tc.ResourceParentRefs[pcKind] = append(tc.ResourceParentRefs[pcKind], ingressClassKind)

	return nil
}

func (r *ApisixPluginConfigReconciler) updateStatus(pc *apiv2.ApisixPluginConfig, err error) {
	SetApisixPluginConfigConditionAccepted(&pc.Status, pc.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(pc),
		Resource:       &apiv2.ApisixPluginConfig{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp, ok := obj.(*apiv2.ApisixPluginConfig)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			cpCopy := cp.DeepCopy()
			cpCopy.Status = pc.Status
			return cpCopy
		}),
	})
}
