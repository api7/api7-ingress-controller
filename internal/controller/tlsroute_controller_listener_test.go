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
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

type tlsRouteListenerProvider struct {
	lastContext *provider.TranslateContext
}

func (p *tlsRouteListenerProvider) Register(string, *http.ServeMux) {}

func (p *tlsRouteListenerProvider) Update(_ context.Context, tctx *provider.TranslateContext, _ client.Object) error {
	p.lastContext = tctx
	return nil
}

func (p *tlsRouteListenerProvider) Delete(context.Context, client.Object) error { return nil }

func (p *tlsRouteListenerProvider) Start(context.Context) error { return nil }

func (p *tlsRouteListenerProvider) NeedLeaderElection() bool { return true }

func TestTLSRouteReconcilePropagatesMatchedListener(t *testing.T) {
	scheme := parentRefTestScheme(t)
	gatewayClass := newParentRefGatewayClass()
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gatewayClass.Name),
			Listeners: []gatewayv1.Listener{
				{Name: "tls-main", Protocol: gatewayv1.TLSProtocolType, Port: 9110},
				{Name: "tls-sibling", Protocol: gatewayv1.TLSProtocolType, Port: 9111},
			},
		},
	}
	sectionName := gatewayv1.SectionName("tls-main")
	route := &gatewayv1.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Name:        gatewayv1.ObjectName(gateway.Name),
					SectionName: &sectionName,
				}},
			},
			Hostnames: []gatewayv1.Hostname{"example.com"},
		},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects([]client.Object{gatewayClass, gateway, route}...).
		Build()
	prov := &tlsRouteListenerProvider{}
	r := &TLSRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  &recordingUpdater{},
		Readier:  noopReadier{},
	}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: route.Namespace, Name: route.Name},
	})
	require.NoError(t, err)
	require.NotNil(t, prov.lastContext)
	require.Equal(t, []gatewayv1.Listener{gateway.Spec.Listeners[0]}, prov.lastContext.Listeners)
	require.True(t, prov.lastContext.HasExplicitListenerMatch)
}
