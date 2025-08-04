// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/networking/v1beta1"
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
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// IngressClassV1beta1Reconciler reconciles a IngressClassV1beta1 object.
type IngressClassV1beta1Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressClassV1beta1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1beta1.IngressClass{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesController),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
			),
		).
		Watches(
			&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressClassV1beta1esForGatewayProxy),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressClassV1beta1esForSecret),
		).
		Complete(r)
}

func (r *IngressClassV1beta1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	IngressClassV1beta1 := new(v1beta1.IngressClass)
	if err := r.Get(ctx, req.NamespacedName, IngressClassV1beta1); err != nil {
		if client.IgnoreNotFound(err) == nil {
			IngressClassV1beta1.Name = req.Name

			IngressClassV1beta1.TypeMeta = metav1.TypeMeta{
				Kind:       KindIngressClass,
				APIVersion: v1beta1.SchemeGroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, IngressClassV1beta1); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	if err := r.processInfrastructure(tctx, IngressClassV1beta1); err != nil {
		r.Log.Error(err, "failed to process infrastructure for IngressClassV1beta1", "IngressClassV1beta1", IngressClassV1beta1.GetName())
		return ctrl.Result{}, err
	}

	if err := r.Provider.Update(ctx, tctx, IngressClassV1beta1); err != nil {
		r.Log.Error(err, "failed to update IngressClassV1beta1", "IngressClassV1beta1", IngressClassV1beta1.GetName())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IngressClassV1beta1Reconciler) matchesController(obj client.Object) bool {
	IngressClassV1beta1, ok := obj.(*v1beta1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClassV1beta1")
		return false
	}
	return matchesController(IngressClassV1beta1.Spec.Controller)
}

func (r *IngressClassV1beta1Reconciler) listIngressClassV1beta1esForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayProxy")
		return nil
	}
	namespace := gatewayProxy.GetNamespace()
	name := gatewayProxy.GetName()

	IngressClassV1beta1List := &v1beta1.IngressClassList{}
	if err := r.List(ctx, IngressClassV1beta1List, client.MatchingFields{
		indexer.IngressClassParametersRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	recs := make([]reconcile.Request, 0, len(IngressClassV1beta1List.Items))
	for _, IngressClassV1beta1 := range IngressClassV1beta1List.Items {
		if !r.matchesController(&IngressClassV1beta1) {
			continue
		}
		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name: IngressClassV1beta1.GetName(),
			},
		})
	}
	return recs
}

func (r *IngressClassV1beta1Reconciler) listIngressClassV1beta1esForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Secret")
		return nil
	}

	// 1. list gateway proxies by secret
	gatewayProxyList := &v1alpha1.GatewayProxyList{}
	if err := r.List(ctx, gatewayProxyList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list gateway proxies by secret", "secret", secret.GetName())
		return nil
	}

	// 2. list ingress classes by gateway proxies
	requests := make([]reconcile.Request, 0)
	for _, gatewayProxy := range gatewayProxyList.Items {
		requests = append(requests, r.listIngressClassV1beta1esForGatewayProxy(ctx, &gatewayProxy)...)
	}

	return distinctRequests(requests)
}

func (r *IngressClassV1beta1Reconciler) processInfrastructure(tctx *provider.TranslateContext, IngressClassV1beta1 *v1beta1.IngressClass) error {
	if IngressClassV1beta1.Spec.Parameters == nil {
		return nil
	}

	if IngressClassV1beta1.Spec.Parameters.APIGroup == nil ||
		*IngressClassV1beta1.Spec.Parameters.APIGroup != v1alpha1.GroupVersion.Group ||
		IngressClassV1beta1.Spec.Parameters.Kind != KindGatewayProxy {
		return nil
	}

	// Since v1beta1 does not support specifying the target namespace,
	// and GatewayProxy is a namespace-scoped resource, we first check
	// for the annotation, then fall back to the spec field, and finally
	// default to "default" namespace for convenience.
	namespace := "default"
	if IngressClassV1beta1.Spec.Parameters.Namespace != nil {
		namespace = *IngressClassV1beta1.Spec.Parameters.Namespace
	}
	// Check for annotation override
	if annotationNamespace, exists := IngressClassV1beta1.Annotations[gatewayProxyNamespaceAnnotation]; exists && annotationNamespace != "" {
		namespace = annotationNamespace
	}

	gatewayProxy := new(v1alpha1.GatewayProxy)
	if err := r.Get(tctx, client.ObjectKey{
		Namespace: namespace,
		Name:      IngressClassV1beta1.Spec.Parameters.Name,
	}, gatewayProxy); err != nil {
		return fmt.Errorf("failed to get gateway proxy: %w", err)
	}

	rk := utils.NamespacedNameKind(IngressClassV1beta1)

	tctx.GatewayProxies[rk] = *gatewayProxy
	tctx.ResourceParentRefs[rk] = append(tctx.ResourceParentRefs[rk], rk)

	// Load secrets if needed
	if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.ControlPlane != nil {
		auth := gatewayProxy.Spec.Provider.ControlPlane.Auth
		if auth.Type == v1alpha1.AuthTypeAdminKey && auth.AdminKey != nil && auth.AdminKey.ValueFrom != nil {
			if auth.AdminKey.ValueFrom.SecretKeyRef != nil {
				secretRef := auth.AdminKey.ValueFrom.SecretKeyRef
				secret := &corev1.Secret{}
				if err := r.Get(tctx, client.ObjectKey{
					Namespace: namespace,
					Name:      secretRef.Name,
				}, secret); err != nil {
					r.Log.Error(err, "failed to get secret for gateway proxy", "namespace", namespace, "name", secretRef.Name)
					return err
				}
				tctx.Secrets[client.ObjectKey{
					Namespace: namespace,
					Name:      secretRef.Name,
				}] = secret
			}
		}
	}

	if service := gatewayProxy.Spec.Provider.ControlPlane.Service; service != nil {
		if err := addProviderEndpointsToTranslateContext(tctx, r.Client, types.NamespacedName{
			Namespace: gatewayProxy.GetNamespace(),
			Name:      service.Name,
		}); err != nil {
			return err
		}
	}

	_, ok := tctx.GatewayProxies[rk]
	if !ok {
		return fmt.Errorf("no gateway proxy found for ingress class")
	}

	return nil
}
