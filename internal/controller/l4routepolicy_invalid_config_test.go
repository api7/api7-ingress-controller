// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestL4RouteReconcileRejectsInvalidPolicyPluginConfig(t *testing.T) {
	const (
		namespace  = "default"
		routeName  = "route"
		policyName = "policy"
	)

	tests := []struct {
		name       string
		routeKind  gatewayv1.Kind
		protocol   gatewayv1.ProtocolType
		newRoute   func() client.Object
		reconcile  func(client.Client, provider.Provider, status.Updater) error
		parentRefs func(client.Object) []gatewayv1.RouteParentStatus
	}{
		{
			name:      "TCPRoute",
			routeKind: "TCPRoute",
			protocol:  gatewayv1.TCPProtocolType,
			newRoute: func() client.Object {
				return &gatewayv1.TCPRoute{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: routeName},
					Spec: gatewayv1.TCPRouteSpec{
						CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gw"}}},
						Rules:           []gatewayv1.TCPRouteRule{{}},
					},
				}
			},
			reconcile: func(cli client.Client, prov provider.Provider, updater status.Updater) error {
				r := &TCPRouteReconciler{Client: cli, Log: logr.Discard(), Provider: prov, Updater: updater, Readier: noopReadier{}, supportsL4RoutePolicy: true}
				_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: namespace, Name: routeName}})
				return err
			},
			parentRefs: func(obj client.Object) []gatewayv1.RouteParentStatus {
				return obj.(*gatewayv1.TCPRoute).Status.Parents
			},
		},
		{
			name:      "UDPRoute",
			routeKind: "UDPRoute",
			protocol:  gatewayv1.UDPProtocolType,
			newRoute: func() client.Object {
				return &gatewayv1.UDPRoute{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: routeName},
					Spec: gatewayv1.UDPRouteSpec{
						CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gw"}}},
						Rules:           []gatewayv1.UDPRouteRule{{}},
					},
				}
			},
			reconcile: func(cli client.Client, prov provider.Provider, updater status.Updater) error {
				r := &UDPRouteReconciler{Client: cli, Log: logr.Discard(), Provider: prov, Updater: updater, Readier: noopReadier{}, supportsL4RoutePolicy: true}
				_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: namespace, Name: routeName}})
				return err
			},
			parentRefs: func(obj client.Object) []gatewayv1.RouteParentStatus {
				return obj.(*gatewayv1.UDPRoute).Status.Parents
			},
		},
		{
			name:      "TLSRoute",
			routeKind: "TLSRoute",
			protocol:  gatewayv1.TLSProtocolType,
			newRoute: func() client.Object {
				return &gatewayv1.TLSRoute{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: routeName},
					Spec: gatewayv1.TLSRouteSpec{
						CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gw"}}},
						Hostnames:       []gatewayv1.Hostname{"example.com"},
						Rules:           []gatewayv1.TLSRouteRule{{}},
					},
				}
			},
			reconcile: func(cli client.Client, prov provider.Provider, updater status.Updater) error {
				r := &TLSRouteReconciler{Client: cli, Log: logr.Discard(), Provider: prov, Updater: updater, Readier: noopReadier{}, supportsL4RoutePolicy: true}
				_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: namespace, Name: routeName}})
				return err
			},
			parentRefs: func(obj client.Object) []gatewayv1.RouteParentStatus {
				return obj.(*gatewayv1.TLSRoute).Status.Parents
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, gatewayv1.Install(scheme))
			require.NoError(t, v1alpha1.AddToScheme(scheme))

			gatewayClass := &gatewayv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
				Spec: gatewayv1.GatewayClassSpec{
					ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
				},
			}
			gateway := &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "gw"},
				Spec: gatewayv1.GatewaySpec{
					GatewayClassName: "apisix",
					Listeners: []gatewayv1.Listener{{
						Name:     "listener",
						Protocol: tt.protocol,
						Port:     9000,
					}},
				},
			}
			route := tt.newRoute()
			policy := &v1alpha1.L4RoutePolicy{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: policyName, Generation: 1},
				Spec: v1alpha1.L4RoutePolicySpec{
					TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
						LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
							Group: gatewayv1.GroupName,
							Kind:  tt.routeKind,
							Name:  routeName,
						},
					}},
					Plugins: []v1alpha1.Plugin{{
						Name:   "ip-restriction",
						Config: apiextensionsv1.JSON{Raw: []byte(`["should-not-appear"]`)},
					}},
				},
			}

			cli := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(gatewayClass, gateway, route, policy).
				WithIndex(&v1alpha1.L4RoutePolicy{}, indexer.PolicyTargetRefs, indexer.L4RoutePolicyIndexFunc).
				Build()
			prov := &recordingProvider{}
			updater := &recordingUpdater{}

			require.NoError(t, tt.reconcile(cli, prov, updater))
			assert.Zero(t, prov.updated, "an invalid policy must keep the existing provider state")
			assert.Empty(t, prov.deleted)

			var gotPolicyStatus *v1alpha1.L4RoutePolicy
			var gotRouteStatus client.Object
			for _, update := range updater.updates {
				switch update.Resource.(type) {
				case *v1alpha1.L4RoutePolicy:
					gotPolicyStatus = update.Mutator.Mutate(policy.DeepCopy()).(*v1alpha1.L4RoutePolicy)
				default:
					gotRouteStatus = update.Mutator.Mutate(route.DeepCopyObject().(client.Object))
				}
			}

			require.NotNil(t, gotPolicyStatus)
			require.Len(t, gotPolicyStatus.Status.Ancestors, 1)
			require.Len(t, gotPolicyStatus.Status.Ancestors[0].Conditions, 1)
			condition := gotPolicyStatus.Status.Ancestors[0].Conditions[0]
			assert.Equal(t, metav1.ConditionFalse, condition.Status)
			assert.Equal(t, string(gatewayv1.PolicyReasonInvalid), condition.Reason)
			assert.Contains(t, condition.Message, `plugin "ip-restriction" has an invalid configuration`)
			assert.NotContains(t, condition.Message, "should-not-appear")

			require.NotNil(t, gotRouteStatus)
			parents := tt.parentRefs(gotRouteStatus)
			require.Len(t, parents, 1)
			var accepted *metav1.Condition
			for i := range parents[0].Conditions {
				if parents[0].Conditions[i].Type == string(gatewayv1.RouteConditionAccepted) {
					accepted = &parents[0].Conditions[i]
					break
				}
			}
			require.NotNil(t, accepted)
			assert.Equal(t, metav1.ConditionTrue, accepted.Status)
		})
	}
}
