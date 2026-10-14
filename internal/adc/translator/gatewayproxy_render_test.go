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

package translator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func TestGatewayProxyPluginRenderErrors(t *testing.T) {
	badJSON := apiextensionsv1.JSON{Raw: []byte(`["not-an-object"]`)}

	objects := []struct {
		name      string
		object    client.Object
		translate func(*Translator, *provider.TranslateContext, client.Object) (*TranslateResult, error)
	}{
		{
			name: "Gateway",
			object: &gatewayv1.Gateway{
				TypeMeta:   metav1.TypeMeta{APIVersion: gatewayv1.GroupVersion.String(), Kind: "Gateway"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gateway"},
			},
			translate: func(tr *Translator, tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
				return tr.TranslateGateway(tctx, obj.(*gatewayv1.Gateway))
			},
		},
		{
			name: "IngressClass",
			object: &networkingv1.IngressClass{
				TypeMeta:   metav1.TypeMeta{APIVersion: networkingv1.SchemeGroupVersion.String(), Kind: "IngressClass"},
				ObjectMeta: metav1.ObjectMeta{Name: "ingress-class"},
			},
			translate: func(tr *Translator, tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
				return tr.TranslateIngressClass(tctx, obj.(*networkingv1.IngressClass))
			},
		},
	}

	tests := []struct {
		name      string
		proxy     v1alpha1.GatewayProxy
		wantError string
	}{
		{
			name: "plugin config",
			proxy: v1alpha1.GatewayProxy{Spec: v1alpha1.GatewayProxySpec{
				Plugins: []v1alpha1.GatewayProxyPlugin{{
					Name: "response-rewrite", Enabled: true, Config: badJSON,
				}},
			}},
			wantError: `failed to unmarshal config of GatewayProxy plugin "response-rewrite"`,
		},
		{
			name: "plugin metadata",
			proxy: v1alpha1.GatewayProxy{Spec: v1alpha1.GatewayProxySpec{
				PluginMetadata: map[string]apiextensionsv1.JSON{"key-auth": badJSON},
			}},
			wantError: `failed to unmarshal GatewayProxy plugin metadata for "key-auth"`,
		},
	}

	for _, object := range objects {
		for _, tt := range tests {
			t.Run(object.name+"/"+tt.name, func(t *testing.T) {
				tctx := provider.NewDefaultTranslateContext(context.Background())
				tctx.GatewayProxies[utils.NamespacedNameKind(object.object)] = tt.proxy

				result, err := object.translate(&Translator{Log: logr.Discard()}, tctx, object.object)
				require.Nil(t, result)
				require.ErrorContains(t, err, tt.wantError)
			})
		}
	}
}
