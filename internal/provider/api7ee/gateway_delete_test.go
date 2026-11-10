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

package api7ee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

type gatewayDeleteReadier struct{ readiness.ReadinessManager }

func (gatewayDeleteReadier) Done(client.Object, k8stypes.NamespacedName) {}

func TestGatewayDeletionRetractsRouteWithUnknownSibling(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	}))
	defer server.Close()
	t.Setenv("ADC_SERVER_URL", server.URL)
	adc, err := adcclient.New(logr.Discard(), "api7ee", time.Second)
	require.NoError(t, err)
	d := &api7eeProvider{client: adc, syncCh: make(chan struct{}, 1), log: logr.Discard()}

	gateway := &gatewayv1.Gateway{
		TypeMeta:   metav1.TypeMeta{Kind: "Gateway", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "deleted"},
	}
	route := &gatewayv1.HTTPRoute{
		TypeMeta:   metav1.TypeMeta{Kind: "HTTPRoute", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{{Name: "deleted"}, {Name: "unresolved"}},
		}},
	}
	sibling := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "unresolved"},
		Spec:       gatewayv1.GatewaySpec{GatewayClassName: "missing-class"},
	}
	proxy := types.NamespacedNameKind{Kind: "GatewayProxy", Namespace: "default", Name: "proxy"}
	configs := map[types.NamespacedNameKind]adctypes.Config{proxy: {Name: proxy.String(), BackendType: "apisix", ServerAddrs: []string{"http://apisix:9180"}}}

	for _, obj := range []client.Object{gateway, route} {
		resources := &adctypes.Resources{}
		resourceType := adctypes.TypeSSL
		if obj == route {
			resourceType = adctypes.TypeService
			resources.Services = []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "stale-route", Labels: label.GenLabel(route)}}}
		}
		require.NoError(t, adc.UpdateConfig(context.Background(), adcclient.Task{
			Key: utils.NamespacedNameKind(obj), Configs: configs, ResourceTypes: []string{resourceType}, Resources: resources, Labels: label.GenLabel(obj),
		}))
	}

	require.NoError(t, d.Delete(context.Background(), gateway))
	scheme := runtime.NewScheme()
	require.NoError(t, gatewayv1.Install(scheme))
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(route, sibling).Build()
	r := &controller.HTTPRouteReconciler{Client: cli, Scheme: scheme, Log: logr.Discard(), Provider: d, Readier: gatewayDeleteReadier{}}
	_, err = r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(route)})
	require.NoError(t, err)
	resources, err := adc.GetResources(proxy.String())
	require.NoError(t, err)
	require.Empty(t, resources.Services, "deleted Gateway's route survives reconciliation with an unresolved sibling")
}
