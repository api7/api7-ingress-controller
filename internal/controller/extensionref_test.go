// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestRouteExtensionRefResolutionStatus(t *testing.T) {
	validGroup := gatewayv1.Group(v1alpha1.GroupVersion.Group)
	tests := []struct {
		name       string
		ref        gatewayv1.LocalObjectReference
		objects    []client.Object
		wantStatus metav1.ConditionStatus
		wantReason gatewayv1.RouteConditionReason
	}{
		{
			name: "supported reference",
			ref: gatewayv1.LocalObjectReference{
				Group: validGroup,
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			objects: []client.Object{&v1alpha1.PluginConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "filter"},
			}},
			wantStatus: metav1.ConditionTrue,
			wantReason: gatewayv1.RouteReasonResolvedRefs,
		},
		{
			name: "unsupported group",
			ref: gatewayv1.LocalObjectReference{
				Group: "example.com",
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.RouteReasonInvalidKind,
		},
		{
			name: "unsupported kind",
			ref: gatewayv1.LocalObjectReference{
				Group: validGroup,
				Kind:  "OtherFilter",
				Name:  "filter",
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.RouteReasonInvalidKind,
		},
		{
			name: "missing PluginConfig",
			ref: gatewayv1.LocalObjectReference{
				Group: validGroup,
				Kind:  internaltypes.KindPluginConfig,
				Name:  "missing",
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.RouteReasonBackendNotFound,
		},
		{
			name: "missing plugin Secret",
			ref: gatewayv1.LocalObjectReference{
				Group: validGroup,
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			objects: []client.Object{&v1alpha1.PluginConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "filter"},
				Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{
					Name:      "openid-connect",
					SecretRef: &corev1.LocalObjectReference{Name: "credentials"},
				}}},
			}},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.RouteReasonBackendNotFound,
		},
	}

	for _, tt := range tests {
		for _, routeKind := range []string{"HTTPRoute", "GRPCRoute"} {
			t.Run(tt.name+"/"+routeKind, func(t *testing.T) {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, v1alpha1.AddToScheme(scheme))
				cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
				tctx := provider.NewDefaultTranslateContext(context.Background())

				var err error
				if routeKind == "HTTPRoute" {
					route := &gatewayv1.HTTPRoute{
						ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
						Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
							Filters: []gatewayv1.HTTPRouteFilter{{
								Type:         gatewayv1.HTTPRouteFilterExtensionRef,
								ExtensionRef: &tt.ref,
							}},
						}}},
					}
					err = (&HTTPRouteReconciler{Client: cli}).processHTTPRoute(tctx, route)
				} else {
					route := &gatewayv1.GRPCRoute{
						ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
						Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{{
							Filters: []gatewayv1.GRPCRouteFilter{{
								Type:         gatewayv1.GRPCRouteFilterExtensionRef,
								ExtensionRef: &tt.ref,
							}},
						}}},
					}
					err = (&GRPCRouteReconciler{Client: cli}).processGRPCRoute(tctx, route)
				}

				parentStatus := gatewayv1.RouteParentStatus{}
				SetRouteConditionResolvedRefs(&parentStatus, 1, err)
				require.Len(t, parentStatus.Conditions, 1)
				assert.Equal(t, tt.wantStatus, parentStatus.Conditions[0].Status)
				assert.Equal(t, string(tt.wantReason), parentStatus.Conditions[0].Reason)
			})
		}
	}
}
