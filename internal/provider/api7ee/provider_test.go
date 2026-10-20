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
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func TestDeleteLogsObjectIdentityOnly(t *testing.T) {
	const privateValue = "delete-log-private-value"
	var entries []string
	log := funcr.New(func(prefix, args string) {
		entries = append(entries, prefix+args)
	}, funcr.Options{Verbosity: 10})

	p, err := New(log, nil, nil)
	require.NoError(t, err)
	consumer := &apiv2.ApisixConsumer{
		TypeMeta: metav1.TypeMeta{Kind: "ApisixConsumer", APIVersion: apiv2.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "consumer",
		},
		Spec: apiv2.ApisixConsumerSpec{
			AuthParameter: &apiv2.ApisixConsumerAuthParameter{
				KeyAuth: &apiv2.ApisixConsumerKeyAuth{Value: &apiv2.ApisixConsumerKeyAuthValue{Key: privateValue}},
			},
		},
	}

	require.NoError(t, p.Delete(context.Background(), consumer))

	output := strings.Join(entries, "\n")
	assert.NotContains(t, output, privateValue)
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "consumer")
}

func TestUpdateKeepsLastKnownGoodStateWhenL4PolicyCannotRender(t *testing.T) {
	rawProvider, err := New(logr.Discard(), nil, nil)
	require.NoError(t, err)
	d := rawProvider.(*api7eeProvider)

	route := &gatewayv1.TCPRoute{
		TypeMeta: metav1.TypeMeta{Kind: "TCPRoute", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "tcp-route",
		},
		Spec: gatewayv1.TCPRouteSpec{Rules: []gatewayv1.TCPRouteRule{{}}},
	}
	gatewayProxy := v1alpha1.GatewayProxy{
		TypeMeta:   metav1.TypeMeta{Kind: "GatewayProxy", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "proxy"},
		Spec: v1alpha1.GatewayProxySpec{Provider: &v1alpha1.GatewayProxyProvider{
			Type: v1alpha1.ProviderTypeControlPlane,
			ControlPlane: &v1alpha1.ControlPlaneProvider{
				Endpoints: []string{"http://apisix:9180"},
				Auth: v1alpha1.ControlPlaneAuth{
					Type:     v1alpha1.AuthTypeAdminKey,
					AdminKey: &v1alpha1.AdminKeyAuth{Value: "key"},
				},
			},
		}},
	}
	configName := utils.NamespacedNameKind(&gatewayProxy).String()
	lastKnownGood := adctypes.NewDefaultService()
	lastKnownGood.Name = "last-known-good"
	lastKnownGood.ID = "last-known-good"
	lastKnownGood.Labels = label.GenLabel(route)
	require.NoError(t, d.client.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{lastKnownGood},
	}, lastKnownGood.Labels))

	policy := &v1alpha1.L4RoutePolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "tcp-policy"},
		Spec: v1alpha1.L4RoutePolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: gatewayv1.GroupName,
					Kind:  "TCPRoute",
					Name:  "tcp-route",
				},
			}},
			Plugins: []v1alpha1.Plugin{{
				Name:   "ip-restriction",
				Config: apiextensionsv1.JSON{Raw: []byte(`[]`)},
			}},
		},
	}
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.GatewayProxies[utils.NamespacedNameKind(&gatewayProxy)] = gatewayProxy
	tctx.L4RoutePolicies[k8stypes.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = policy

	err = d.Update(context.Background(), tctx, route)

	require.Error(t, err)
	resources, getErr := d.client.GetResources(configName)
	require.NoError(t, getErr)
	require.Len(t, resources.Services, 1)
	assert.Equal(t, "last-known-good", resources.Services[0].Name)
}

func TestDeleteGatewayPreservesSharedProxyResources(t *testing.T) {
	var received adcclient.ADCServerRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	}))
	defer server.Close()
	t.Setenv("ADC_SERVER_URL", server.URL)
	cli, err := adcclient.New(logr.Discard(), "api7ee", time.Second)
	require.NoError(t, err)
	d := &api7eeProvider{client: cli, syncCh: make(chan struct{}, 1), log: logr.Discard()}
	gateway := &gatewayv1.Gateway{
		TypeMeta:   metav1.TypeMeta{Kind: "Gateway", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "deleted"},
	}
	other := gateway.DeepCopy()
	other.Name = "remaining"
	route := &gatewayv1.HTTPRoute{
		TypeMeta:   metav1.TypeMeta{Kind: "HTTPRoute", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}
	proxy := types.NamespacedNameKind{Kind: "GatewayProxy", Namespace: "default", Name: "proxy"}
	configs := map[types.NamespacedNameKind]adctypes.Config{proxy: {
		Name: proxy.String(), BackendType: "apisix", ServerAddrs: []string{"http://apisix:9180"},
	}}
	for _, gw := range []*gatewayv1.Gateway{gateway, other} {
		labels := label.GenLabel(gw)
		require.NoError(t, cli.UpdateConfig(context.Background(), adcclient.Task{
			Key: utils.NamespacedNameKind(gw), Configs: configs, Labels: labels, ResourceTypes: []string{adctypes.TypeSSL},
			Resources: &adctypes.Resources{SSLs: []*adctypes.SSL{{Metadata: adctypes.Metadata{ID: gw.Name, Labels: labels}}}},
		}))
	}
	labels := label.GenLabel(route)
	require.NoError(t, cli.UpdateConfig(context.Background(), adcclient.Task{
		Key: utils.NamespacedNameKind(route), Configs: configs, Labels: labels, ResourceTypes: []string{adctypes.TypeService},
		Resources: &adctypes.Resources{Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "route", Labels: labels}}}},
	}))
	require.NoError(t, d.Delete(context.Background(), gateway))
	resources, err := cli.GetResources(proxy.String())
	require.NoError(t, err)
	require.Len(t, resources.SSLs, 1)
	assert.Equal(t, "remaining", resources.SSLs[0].ID)
	require.Len(t, resources.Services, 1)
	assert.Equal(t, "route", resources.Services[0].ID)
	assert.Equal(t, []string{adctypes.TypeSSL}, received.Task.Opts.IncludeResourceType)
	require.Len(t, d.syncCh, 1, "Gateway deletion must schedule a full sync for the evicted plugins")
	assert.Equal(t, map[string]string{label.LabelResourceKey: label.GenLabel(gateway)[label.LabelResourceKey]}, received.Task.Opts.LabelSelector)
	_, err = cli.Sync(context.Background())
	require.NoError(t, err)
	assert.Equal(t, resources.SSLs, received.Task.Config.SSLs)
	assert.Equal(t, resources.Services, received.Task.Config.Services)
	assert.Empty(t, cli.ConfigManager.Get(utils.NamespacedNameKind(gateway)))
	assert.NotEmpty(t, cli.ConfigManager.Get(utils.NamespacedNameKind(other)))
	assert.NotEmpty(t, cli.ConfigManager.Get(utils.NamespacedNameKind(route)))
}

func TestStartConsumesSyncNotificationsWithoutSyncPeriod(t *testing.T) {
	requests := make(chan struct{}, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
		requests <- struct{}{}
	}))
	defer server.Close()
	t.Setenv("ADC_SERVER_URL", server.URL)
	cli, err := adcclient.New(logr.Discard(), "api7ee", time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readier := readiness.NewReadinessManager(nil, logr.Discard())
	require.NoError(t, readier.Start(ctx))
	d := &api7eeProvider{
		client: cli, readier: readier, updater: discardUpdater{}, syncCh: make(chan struct{}, 1), log: logr.Discard(),
	}

	route := &gatewayv1.HTTPRoute{
		TypeMeta:   metav1.TypeMeta{Kind: "HTTPRoute", APIVersion: gatewayv1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}
	proxy := types.NamespacedNameKind{Kind: "GatewayProxy", Namespace: "default", Name: "proxy"}
	labels := label.GenLabel(route)
	require.NoError(t, cli.UpdateConfig(ctx, adcclient.Task{
		Key: utils.NamespacedNameKind(route), Labels: labels, ResourceTypes: []string{adctypes.TypeService},
		Configs: map[types.NamespacedNameKind]adctypes.Config{proxy: {
			Name: proxy.String(), BackendType: "apisix", ServerAddrs: []string{"http://apisix:9180"},
		}},
		Resources: &adctypes.Resources{Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "route", Labels: labels}}}},
	}))

	done := make(chan error, 1)
	go func() { done <- d.Start(ctx) }()

	select {
	case <-requests:
	case <-time.After(5 * time.Second):
		t.Fatal("startup sync did not run")
	}
	d.syncNotify()
	select {
	case <-requests:
	case err := <-done:
		t.Fatalf("Start returned before consuming the notification: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("sync notification was not consumed")
	}
	cancel()
	require.NoError(t, <-done)
}

type discardUpdater struct{}

func (discardUpdater) Update(status.Update) {}
