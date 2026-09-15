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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func TestApisixConsumerReconcile_RemovesConfigAfterIngressClassHandoff(t *testing.T) {
	const (
		namespace        = "default"
		name             = "consumer"
		managedClass     = "apisix"
		unmanagedClass   = "other"
		unmanagedControl = "example.com/other-controller"
	)

	consumerKey := k8stypes.NamespacedName{Namespace: namespace, Name: name}

	for _, tc := range []struct {
		name           string
		apiVersion     schema.GroupVersion
		ingressClasses []client.Object
	}{
		{
			name:       "v1",
			apiVersion: networkingv1.SchemeGroupVersion,
			ingressClasses: []client.Object{
				&networkingv1.IngressClass{
					ObjectMeta: metav1.ObjectMeta{Name: managedClass},
					Spec:       networkingv1.IngressClassSpec{Controller: config.GetControllerName()},
				},
				&networkingv1.IngressClass{
					ObjectMeta: metav1.ObjectMeta{Name: unmanagedClass},
					Spec:       networkingv1.IngressClassSpec{Controller: unmanagedControl},
				},
			},
		},
		{
			name:       "v1beta1",
			apiVersion: networkingv1beta1.SchemeGroupVersion,
			ingressClasses: []client.Object{
				&networkingv1beta1.IngressClass{
					ObjectMeta: metav1.ObjectMeta{Name: managedClass},
					Spec: networkingv1beta1.IngressClassSpec{
						Controller: config.GetControllerName(),
						Parameters: &networkingv1beta1.IngressClassParametersReference{},
					},
				},
				&networkingv1beta1.IngressClass{
					ObjectMeta: metav1.ObjectMeta{Name: unmanagedClass},
					Spec: networkingv1beta1.IngressClassSpec{
						Controller: unmanagedControl,
						Parameters: &networkingv1beta1.IngressClassParametersReference{},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, clientgoscheme.AddToScheme(scheme))
			require.NoError(t, apiv2.AddToScheme(scheme))

			oldConsumer := &apiv2.ApisixConsumer{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
				Spec:       apiv2.ApisixConsumerSpec{IngressClassName: managedClass},
			}
			consumer := oldConsumer.DeepCopy()
			consumer.Spec.IngressClassName = unmanagedClass

			objects := append([]client.Object{consumer}, tc.ingressClasses...)
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			prov := &recordingProvider{}
			r := &ApisixConsumerReconciler{
				Client:   cli,
				Scheme:   scheme,
				Log:      logr.Discard(),
				Provider: prov,
				Readier:  noopReadier{},
				ICGV:     tc.apiVersion,
			}

			predicate := MatchesIngressClassPredicate(cli, logr.Discard(), tc.apiVersion.String())
			assert.True(t, predicate.Update(event.UpdateEvent{ObjectOld: oldConsumer, ObjectNew: consumer}),
				"the handoff update must be reconciled")

			result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: consumerKey})

			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)
			assert.Equal(t, []k8stypes.NamespacedName{consumerKey}, prov.deleted)
			assert.Zero(t, prov.updated)
		})
	}
}

func TestApisixConsumerReconcile_KeepsConfigOnIngressClassReadError(t *testing.T) {
	const (
		namespace = "default"
		name      = "consumer"
	)
	consumerKey := k8stypes.NamespacedName{Namespace: namespace, Name: name}

	for _, tc := range []struct {
		name       string
		apiVersion schema.GroupVersion
		failGet    func(client.Object) bool
	}{
		{
			name:       "v1",
			apiVersion: networkingv1.SchemeGroupVersion,
			failGet: func(obj client.Object) bool {
				_, ok := obj.(*networkingv1.IngressClass)
				return ok
			},
		},
		{
			name:       "v1beta1",
			apiVersion: networkingv1beta1.SchemeGroupVersion,
			failGet: func(obj client.Object) bool {
				_, ok := obj.(*networkingv1beta1.IngressClass)
				return ok
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, clientgoscheme.AddToScheme(scheme))
			require.NoError(t, apiv2.AddToScheme(scheme))

			consumer := &apiv2.ApisixConsumer{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
				Spec:       apiv2.ApisixConsumerSpec{IngressClassName: "apisix"},
			}
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(consumer).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, cli client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						if tc.failGet(obj) {
							return k8serrors.NewInternalError(errors.New("boom"))
						}
						return cli.Get(ctx, key, obj, opts...)
					},
				}).Build()
			prov := &recordingProvider{}
			r := &ApisixConsumerReconciler{
				Client:   cli,
				Scheme:   scheme,
				Log:      logr.Discard(),
				Provider: prov,
				Readier:  noopReadier{},
				ICGV:     tc.apiVersion,
			}

			_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: consumerKey})

			require.Error(t, err)
			assert.True(t, k8serrors.IsInternalError(err), "want an internal error, got %v", err)
			assert.Empty(t, prov.deleted)
			assert.Zero(t, prov.updated)
		})
	}
}
