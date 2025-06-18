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
	"errors"
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// ApisixRouteReconciler reconciles a ApisixRoute object
type ApisixRouteReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixRoute{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					_, ok := obj.(*corev1.Secret)
					return ok
				}),
			),
		).
		Watches(&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApiRouteForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRouteForGatewayProxy),
		).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForService),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForSecret),
		).
		Watches(&apiv2.ApisixPluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForPluginConfig),
		).
		Named("apisixroute").
		Complete(r)
}

func (r *ApisixRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ar apiv2.ApisixRoute
	if err := r.Get(ctx, req.NamespacedName, &ar); err != nil {
		if client.IgnoreNotFound(err) == nil {
			ar.Namespace = req.Namespace
			ar.Name = req.Name
			ar.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixRoute,
				APIVersion: apiv2.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, &ar); err != nil {
				r.Log.Error(err, "failed to delete apisixroute", "apisixroute", ar)
				return ctrl.Result{}, err
			}
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
		r.updateStatus(&ar, err)
	}()

	if ic, err = r.getIngressClass(&ar); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processIngressClassParameters(ctx, tctx, &ar, ic); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processApisixRoute(ctx, tctx, &ar); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.Provider.Update(ctx, tctx, &ar); err != nil {
		err = ReasonError{
			Reason:  string(apiv2.ConditionReasonSyncFailed),
			Message: err.Error(),
		}
		r.Log.Error(err, "failed to process", "apisixroute", ar)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ApisixRouteReconciler) processApisixRoute(ctx context.Context, tc *provider.TranslateContext, in *apiv2.ApisixRoute) error {
	var (
		rules = make(map[string]struct{})
	)
	for httpIndex, http := range in.Spec.HTTP {
		// check rule names
		if _, ok := rules[http.Name]; ok {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: "duplicate route rule name",
			}
		}
		rules[http.Name] = struct{}{}

		// check plugin config reference
		if http.PluginConfigName != "" {
			pcNamespace := in.Namespace
			if http.PluginConfigNamespace != "" {
				pcNamespace = http.PluginConfigNamespace
			}
			var (
				pc = apiv2.ApisixPluginConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      http.PluginConfigName,
						Namespace: pcNamespace,
					},
				}
				pcNN = utils.NamespacedName(&pc)
			)
			if err := r.Get(ctx, pcNN, &pc); err != nil {
				return ReasonError{
					Reason:  string(apiv2.ConditionReasonInvalidSpec),
					Message: fmt.Sprintf("failed to get ApisixPluginConfig: %s", pcNN),
				}
			}

			// Check if ApisixPluginConfig has IngressClassName and if it matches
			if in.Spec.IngressClassName != pc.Spec.IngressClassName && pc.Spec.IngressClassName != "" {
				var pcIC networkingv1.IngressClass
				if err := r.Get(ctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &pcIC); err != nil {
					return ReasonError{
						Reason:  string(apiv2.ConditionReasonInvalidSpec),
						Message: fmt.Sprintf("failed to get IngressClass %s for ApisixPluginConfig %s: %v", pc.Spec.IngressClassName, pcNN, err),
					}
				}
				if !matchesController(pcIC.Spec.Controller) {
					return ReasonError{
						Reason:  string(apiv2.ConditionReasonInvalidSpec),
						Message: fmt.Sprintf("ApisixPluginConfig %s references IngressClass %s with non-matching controller", pcNN, pc.Spec.IngressClassName),
					}
				}
			}

			tc.ApisixPluginConfigs[pcNN] = &pc

			// Also check secrets referenced by plugin config
			for _, plugin := range pc.Spec.Plugins {
				if !plugin.Enable || plugin.Config == nil || plugin.SecretRef == "" {
					continue
				}
				var (
					secret = corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      plugin.SecretRef,
							Namespace: pc.Namespace,
						},
					}
					secretNN = utils.NamespacedName(&secret)
				)
				if err := r.Get(ctx, secretNN, &secret); err != nil {
					return ReasonError{
						Reason:  string(apiv2.ConditionReasonInvalidSpec),
						Message: fmt.Sprintf("failed to get Secret: %s", secretNN),
					}
				}
				tc.Secrets[secretNN] = &secret
			}
		}

		// check secret
		for _, plugin := range http.Plugins {
			if !plugin.Enable || plugin.Config == nil || plugin.SecretRef == "" {
				continue
			}
			var (
				secret = corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      plugin.SecretRef,
						Namespace: in.Namespace,
					},
				}
				secretNN = utils.NamespacedName(&secret)
			)
			if err := r.Get(ctx, secretNN, &secret); err != nil {
				return ReasonError{
					Reason:  string(apiv2.ConditionReasonInvalidSpec),
					Message: fmt.Sprintf("failed to get Secret: %s", secretNN),
				}
			}

			tc.Secrets[utils.NamespacedName(&secret)] = &secret
		}

		// check vars
		// todo: cache the result to tctx
		if _, err := http.Match.NginxVars.ToVars(); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf(".spec.http[%d].match.exprs: %s", httpIndex, err.Error()),
			}
		}

		// validate remote address
		if err := utils.ValidateRemoteAddrs(http.Match.RemoteAddrs); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf(".spec.http[%d].match.remoteAddrs: %s", httpIndex, err.Error()),
			}
		}

		// process backend
		var backends = make(map[types.NamespacedName]struct{})
		for _, backend := range http.Backends {
			var (
				service = corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      backend.ServiceName,
						Namespace: in.Namespace,
					},
				}
				serviceNN = utils.NamespacedName(&service)
			)
			if _, ok := backends[serviceNN]; ok {
				return ReasonError{
					Reason:  string(apiv2.ConditionReasonInvalidSpec),
					Message: fmt.Sprintf("duplicate backend service: %s", serviceNN),
				}
			}
			backends[serviceNN] = struct{}{}

			if err := r.Get(ctx, serviceNN, &service); err != nil {
				if err := client.IgnoreNotFound(err); err == nil {
					r.Log.Error(errors.New("service not found"), "Service", serviceNN)
					continue
				}
				return err
			}
			if service.Spec.Type == corev1.ServiceTypeExternalName {
				tc.Services[serviceNN] = &service
				continue
			}

			if backend.ResolveGranularity == "service" && service.Spec.ClusterIP == "" {
				r.Log.Error(errors.New("service has no ClusterIP"), "Service", serviceNN, "ResolveGranularity", backend.ResolveGranularity)
				continue
			}

			if !slices.ContainsFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Port == int32(backend.ServicePort.IntValue())
			}) {
				r.Log.Error(errors.New("port not found in service"), "Service", serviceNN, "port", backend.ServicePort.String())
				continue
			}
			tc.Services[serviceNN] = &service

			var endpoints discoveryv1.EndpointSliceList
			if err := r.List(ctx, &endpoints,
				client.InNamespace(service.Namespace),
				client.MatchingLabels{
					discoveryv1.LabelServiceName: service.Name,
				},
			); err != nil {
				return ReasonError{
					Reason:  string(apiv2.ConditionReasonInvalidSpec),
					Message: fmt.Sprintf("failed to list endpoint slices: %v", err),
				}
			}
			tc.EndpointSlices[serviceNN] = endpoints.Items
		}
	}

	return nil
}

func (r *ApisixRouteReconciler) listApisixRoutesForService(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return nil
	}

	var (
		namespace   = endpointSlice.GetNamespace()
		serviceName = endpointSlice.Labels[discoveryv1.LabelServiceName]
		arList      apiv2.ApisixRouteList
	)
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(arList.Items))
	for _, ar := range arList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}
	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) listApisixRoutesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	var (
		arList      apiv2.ApisixRouteList
		pcList      apiv2.ApisixPluginConfigList
		allRequests = make([]reconcile.Request, 0)
	)

	// First, find ApisixRoutes that directly reference this secret
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by secret", "secret", secret.Name)
		return nil
	}
	for _, ar := range arList.Items {
		allRequests = append(allRequests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}

	// Second, find ApisixPluginConfigs that reference this secret
	if err := r.List(ctx, &pcList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixpluginconfigs by secret", "secret", secret.Name)
		return nil
	}

	// Then find ApisixRoutes that reference these PluginConfigs
	for _, pc := range pcList.Items {
		var arListForPC apiv2.ApisixRouteList
		if err := r.List(ctx, &arListForPC, client.MatchingFields{
			indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
		}); err != nil {
			r.Log.Error(err, "failed to list apisixroutes by plugin config", "pluginconfig", pc.Name)
			continue
		}
		for _, ar := range arListForPC.Items {
			allRequests = append(allRequests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
		}
	}

	return pkgutils.DedupComparable(allRequests)
}

func (r *ApisixRouteReconciler) listApiRouteForIngressClass(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	ic, ok := object.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	isDefaultIngressClass := IsDefaultIngressClass(ic)
	var arList apiv2.ApisixRouteList
	if err := r.List(ctx, &arList); err != nil {
		return nil
	}
	for _, ar := range arList.Items {
		if ar.Spec.IngressClassName == ic.Name || (isDefaultIngressClass && ar.Spec.IngressClassName == "") {
			requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
		}
	}
	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) listApisixRouteForGatewayProxy(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	gp, ok := object.(*v1alpha1.GatewayProxy)
	if !ok {
		return nil
	}

	var icList networkingv1.IngressClassList
	if err := r.List(ctx, &icList, client.MatchingFields{
		indexer.IngressClassParametersRef: indexer.GenIndexKey(gp.GetNamespace(), gp.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", gp.GetName())
		return nil
	}

	for _, ic := range icList.Items {
		requests = append(requests, r.listApiRouteForIngressClass(ctx, &ic)...)
	}

	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *ApisixRouteReconciler) getIngressClass(ar *apiv2.ApisixRoute) (*networkingv1.IngressClass, error) {
	if ar.Spec.IngressClassName == "" {
		return r.getDefaultIngressClass()
	}

	var ic networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: ar.Spec.IngressClassName}, &ic); err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *ApisixRouteReconciler) getDefaultIngressClass() (*networkingv1.IngressClass, error) {
	var icList networkingv1.IngressClassList
	if err := r.List(context.Background(), &icList, client.MatchingFields{
		indexer.IngressClass: config.GetControllerName(),
	}); err != nil {
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
		Message: "default ingress class not found or dose not match the controller",
	}
}

// processIngressClassParameters processes the IngressClass parameters that reference GatewayProxy
func (r *ApisixRouteReconciler) processIngressClassParameters(ctx context.Context, tc *provider.TranslateContext, ar *apiv2.ApisixRoute, ingressClass *networkingv1.IngressClass) error {
	if ingressClass == nil || ingressClass.Spec.Parameters == nil {
		return nil
	}

	var (
		ingressClassKind = utils.NamespacedNameKind(ingressClass)
		globalRuleKind   = utils.NamespacedNameKind(ar)
		parameters       = ingressClass.Spec.Parameters
	)
	if parameters.APIGroup == nil || *parameters.APIGroup != v1alpha1.GroupVersion.Group || parameters.Kind != KindGatewayProxy {
		return nil
	}

	// check if the parameters reference GatewayProxy
	var (
		gatewayProxy v1alpha1.GatewayProxy
		ns           = *cmp.Or(parameters.Namespace, &ar.Namespace)
	)

	if err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: parameters.Name}, &gatewayProxy); err != nil {
		r.Log.Error(err, "failed to get GatewayProxy", "namespace", ns, "name", parameters.Name)
		return err
	}

	tc.GatewayProxies[ingressClassKind] = gatewayProxy
	tc.ResourceParentRefs[globalRuleKind] = append(tc.ResourceParentRefs[globalRuleKind], ingressClassKind)

	// check if the provider field references a secret
	if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane {
		if gatewayProxy.Spec.Provider.ControlPlane != nil &&
			gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
			gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
			gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
			gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

			secretRef := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
			secret := &corev1.Secret{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: ns,
				Name:      secretRef.Name,
			}, secret); err != nil {
				r.Log.Error(err, "failed to get secret for GatewayProxy provider",
					"namespace", ns,
					"name", secretRef.Name)
				return err
			}

			r.Log.Info("found secret for GatewayProxy provider",
				"ingressClass", ingressClass.Name,
				"gatewayproxy", gatewayProxy.Name,
				"secret", secretRef.Name)

			tc.Secrets[types.NamespacedName{
				Namespace: ns,
				Name:      secretRef.Name,
			}] = secret
		}
	}

	return nil
}

func (r *ApisixRouteReconciler) updateStatus(ar *apiv2.ApisixRoute, err error) {
	SetApisixCRDConditionAccepted(&ar.Status, ar.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(ar),
		Resource:       &apiv2.ApisixRoute{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp := obj.(*apiv2.ApisixRoute).DeepCopy()
			cp.Status = ar.Status
			return cp
		}),
	})
}

func (r *ApisixRouteReconciler) listApisixRoutesForPluginConfig(ctx context.Context, obj client.Object) []reconcile.Request {
	pc, ok := obj.(*apiv2.ApisixPluginConfig)
	if !ok {
		return nil
	}

	// First check if the ApisixPluginConfig has matching IngressClassName
	if pc.Spec.IngressClassName != "" {
		var ic networkingv1.IngressClass
		if err := r.Get(ctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &ic); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get IngressClass for ApisixPluginConfig", "pluginconfig", pc.Name)
			}
			return nil
		}
		if !matchesController(ic.Spec.Controller) {
			return nil
		}
	}

	var arList apiv2.ApisixRouteList
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by plugin config", "pluginconfig", pc.Name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(arList.Items))
	for _, ar := range arList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}
	return pkgutils.DedupComparable(requests)
}
