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
	"cmp"
	"context"

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
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixUpstreamReconciler reconciles a ApisixUpstream object
type ApisixUpstreamReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Log     logr.Logger
	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixUpstreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixUpstream{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixUpstreamForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixUpstreamForGatewayProxy),
		).
		Named("apisixupstream").
		Complete(r)
}

func (r *ApisixUpstreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var au apiv2.ApisixUpstream
	if err := r.Get(ctx, req.NamespacedName, &au); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var (
		ic  *networkingv1.IngressClass
		err error
	)
	defer func() {
		r.updateStatus(&au, err)
	}()

	if ic, err = r.getIngressClass(&au); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processIngressClassParameters(ctx, &au, ic); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ApisixUpstreamReconciler) listApisixUpstreamForIngressClass(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	ic, ok := object.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	isDefaultIngressClass := IsDefaultIngressClass(ic)
	var auList apiv2.ApisixUpstreamList
	if err := r.List(ctx, &auList); err != nil {
		return nil
	}
	for _, pc := range auList.Items {
		if pc.Spec.IngressClassName == ic.Name || (isDefaultIngressClass && pc.Spec.IngressClassName == "") {
			requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&pc)})
		}
	}
	return requests
}

func (r *ApisixUpstreamReconciler) listApisixUpstreamForGatewayProxy(ctx context.Context, object client.Object) (requests []reconcile.Request) {
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
		requests = append(requests, r.listApisixUpstreamForIngressClass(ctx, &ic)...)
	}

	return requests
}

func (r *ApisixUpstreamReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *ApisixUpstreamReconciler) getIngressClass(au *apiv2.ApisixUpstream) (*networkingv1.IngressClass, error) {
	if au.Spec.IngressClassName == "" {
		return r.getDefaultIngressClass()
	}

	var ic networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: au.Spec.IngressClassName}, &ic); err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *ApisixUpstreamReconciler) processIngressClassParameters(ctx context.Context, au *apiv2.ApisixUpstream, ic *networkingv1.IngressClass) error {
	if ic == nil || ic.Spec.Parameters == nil {
		return nil
	}

	var (
		parameters = ic.Spec.Parameters
	)
	if parameters.APIGroup == nil || *parameters.APIGroup != v1alpha1.GroupVersion.Group || parameters.Kind != KindGatewayProxy {
		return nil
	}

	// check if the parameters reference GatewayProxy
	var (
		gp v1alpha1.GatewayProxy
		ns = cmp.Or(parameters.Namespace, &au.Namespace)
	)

	return r.Get(ctx, client.ObjectKey{Namespace: *ns, Name: parameters.Name}, &gp)
}

func (r *ApisixUpstreamReconciler) getDefaultIngressClass() (*networkingv1.IngressClass, error) {
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

func (r *ApisixUpstreamReconciler) updateStatus(au *apiv2.ApisixUpstream, err error) {
	SetApisixCRDConditionAccepted(&au.Status, au.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(au),
		Resource:       &apiv2.ApisixUpstream{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp := obj.(*apiv2.ApisixUpstream).DeepCopy()
			cp.Status = au.Status
			return cp
		}),
	})
}
