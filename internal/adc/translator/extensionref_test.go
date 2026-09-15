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

package translator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestTranslateHTTPRouteUnresolvedExtensionRefReturnsErrorResponsePerRule(t *testing.T) {
	tests := []struct {
		name string
		ref  gatewayv1.LocalObjectReference
	}{
		{
			name: "unsupported group",
			ref: gatewayv1.LocalObjectReference{
				Group: "example.com",
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
		},
		{
			name: "unsupported kind",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  "OtherFilter",
				Name:  "filter",
			},
		},
		{
			name: "missing PluginConfig",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "filter"}] = &v1alpha1.PluginConfig{
				Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{Name: "prometheus"}}},
			}
			route := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
				Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{
					{
						Filters: []gatewayv1.HTTPRouteFilter{{
							Type:         gatewayv1.HTTPRouteFilterExtensionRef,
							ExtensionRef: &tt.ref,
						}},
					},
					{},
				}},
			}

			result, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 2)
			assert.Equal(t, extensionRefErrorResponse(), result.Services[0].Plugins["fault-injection"])
			assert.NotContains(t, result.Services[0].Plugins, "prometheus")
			assert.NotContains(t, result.Services[1].Plugins, "fault-injection")
		})
	}
}

func TestTranslateGRPCRouteUnresolvedExtensionRefReturnsErrorResponsePerRule(t *testing.T) {
	tests := []struct {
		name string
		ref  gatewayv1.LocalObjectReference
	}{
		{
			name: "unsupported reference",
			ref: gatewayv1.LocalObjectReference{
				Group: "example.com",
				Kind:  "OtherFilter",
				Name:  "filter",
			},
		},
		{
			name: "missing PluginConfig",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			route := &gatewayv1.GRPCRoute{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
				Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{
					{
						Filters: []gatewayv1.GRPCRouteFilter{{
							Type:         gatewayv1.GRPCRouteFilterExtensionRef,
							ExtensionRef: &tt.ref,
						}},
					},
					{},
				}},
			}

			result, err := NewTranslator(logr.Discard(), "").TranslateGRPCRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 2)
			assert.Equal(t, extensionRefErrorResponse(), result.Services[0].Plugins["fault-injection"])
			assert.NotContains(t, result.Services[1].Plugins, "fault-injection")
		})
	}
}

func TestTranslateHTTPRouteResolvedExtensionRefAddsPlugins(t *testing.T) {
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "filter"}] = &v1alpha1.PluginConfig{
		Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{Name: "prometheus"}}},
	}
	ref := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
		Kind:  internaltypes.KindPluginConfig,
		Name:  "filter",
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
			Filters: []gatewayv1.HTTPRouteFilter{{
				Type:         gatewayv1.HTTPRouteFilterExtensionRef,
				ExtensionRef: &ref,
			}},
		}}},
	}

	result, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(tctx, route)
	require.NoError(t, err)
	require.Len(t, result.Services, 1)
	assert.Contains(t, result.Services[0].Plugins, "prometheus")
	assert.NotContains(t, result.Services[0].Plugins, "fault-injection")
}

func TestTranslateHTTPRoutePluginConfigWithMissingSecretReturnsErrorResponse(t *testing.T) {
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "filter"}] = &v1alpha1.PluginConfig{
		Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{
			Name:      "openid-connect",
			SecretRef: &corev1.LocalObjectReference{Name: "credentials"},
		}}},
	}
	ref := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
		Kind:  internaltypes.KindPluginConfig,
		Name:  "filter",
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
			Filters: []gatewayv1.HTTPRouteFilter{{
				Type:         gatewayv1.HTTPRouteFilterExtensionRef,
				ExtensionRef: &ref,
			}},
		}}},
	}

	result, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(tctx, route)
	require.NoError(t, err)
	require.Len(t, result.Services, 1)
	assert.Equal(t, extensionRefErrorResponse(), result.Services[0].Plugins["fault-injection"])
}

func extensionRefErrorResponse() map[string]any {
	return map[string]any{
		"abort": map[string]any{
			"http_status": 500,
			"body":        "ExtensionRef filter could not be resolved",
		},
	}
}
