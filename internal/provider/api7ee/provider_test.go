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
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// withMockADCServer starts an ADC server stub and points ADC_SERVER_URL at it for the
// duration of the test. The handler itself is how a test inspects what it received.
func withMockADCServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Setenv("ADC_SERVER_URL", server.URL)
	t.Cleanup(server.Close)
}

// newTestProvider builds a minimally-wired api7eeProvider against the given mock ADC
// server -- every field Delete/sync touch, none of the manager/controller ones.
func newTestProvider(t *testing.T) *api7eeProvider {
	t.Helper()
	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPI7EE, time.Second)
	require.NoError(t, err)
	return &api7eeProvider{
		client:        cli,
		store:         cache.NewStore(logr.Discard()),
		configManager: common.NewConfigManager[types.NamespacedNameKind, adctypes.Config](),
		syncLocks:     common.NewKeyedMutex(),
		syncCh:        make(chan struct{}, 1),
		log:           logr.Discard(),
	}
}

// TestDeletePushesImmediatelyRegardlessOfStartup covers what sets api7ee apart from
// apisix: every Delete pushes right away, whether or not startup synchronization has
// completed -- unlike Update, which defers to the periodic sync until it has.
func TestDeletePushesImmediatelyRegardlessOfStartup(t *testing.T) {
	var mu sync.Mutex
	var received []adcclient.ADCServerRequest

	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)
	// startUpSync is deliberately left false: Delete must not wait for it.

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}
	d.configManager.Update(utils.NamespacedNameKind(route), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {
			Name:        "proxy",
			BackendType: "apisix",
			ServerAddrs: []string{"http://apisix:9080"},
		},
	})

	require.NoError(t, d.Delete(context.Background(), route))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1, "deleting a route must push immediately, not wait for the next scheduled round")
	assert.Equal(t, "proxy", received[0].Task.Opts.CacheKey)
}

// TestSyncStillPushesHealthyConfigsWhenAnotherFails covers sync's error aggregation: one
// GatewayProxy's push failing must not stop the others in the same round from being
// attempted, and the failure must still be reported.
func TestSyncStillPushesHealthyConfigsWhenAnotherFails(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}

	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		seen[req.Task.Opts.CacheKey] = true
		mu.Unlock()
		if req.Task.Opts.CacheKey == "bad" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "boom"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)
	for _, name := range []string{"bad", "good"} {
		key := types.NamespacedNameKind{Namespace: "default", Name: name, Kind: "GatewayProxy"}
		d.configManager.UpdateConfig(key, adctypes.Config{
			Name:        name,
			BackendType: "apisix",
			ServerAddrs: []string{"http://apisix:9080"},
		})
	}

	err := d.sync(context.Background())
	require.Error(t, err, "one config failing must still be reported")
	assert.Contains(t, err.Error(), "bad")

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, seen["bad"], "the failing config must still have been attempted")
	assert.True(t, seen["good"], "a config failing must not stop the others from being pushed")
}

// TestPushConfigsNowMergesGlobalRulesFromStore covers the fork-specific piece the client
// package no longer holds: global_rule is a singleton per config, not partitioned by
// label, so an immediate push scoped to "global_rule" must carry every contribution
// currently in store for that config, not just the one this call is pushing.
func TestPushConfigsNowMergesGlobalRulesFromStore(t *testing.T) {
	var mu sync.Mutex
	var received []adcclient.ADCServerRequest

	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)
	cfg := adctypes.Config{Name: "proxy", BackendType: "apisix", ServerAddrs: []string{"http://apisix:9080"}}

	// Another ApisixGlobalRule already contributed a rule to this same config's store
	// entry before this call.
	require.NoError(t, d.store.Insert(cfg.Name, []string{"global_rule"}, &adctypes.Resources{
		GlobalRules: adctypes.GlobalRule{"limit-count": map[string]any{"count": float64(1)}},
	}, map[string]string{"k8s/resource-key": "ApisixGlobalRule/default/one"}))

	configs := map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: cfg,
	}
	resources := &adctypes.Resources{
		GlobalRules: adctypes.GlobalRule{"key-auth": map[string]any{"key": "k"}},
	}
	labels := map[string]string{"k8s/resource-key": "ApisixGlobalRule/default/two"}

	// pushConfigsNow is the immediate half of Update, called only after applyResourceState
	// has already put this call's own contribution in the store -- mirror that here.
	require.NoError(t, d.store.Insert(cfg.Name, []string{"global_rule"}, resources, labels))

	require.NoError(t, d.pushConfigsNow(context.Background(), configs, []string{"global_rule"}, resources, labels))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1)
	assert.Contains(t, received[0].Task.Config.GlobalRules, "limit-count",
		"the other ApisixGlobalRule's contribution must not be dropped by this push")
	assert.Contains(t, received[0].Task.Config.GlobalRules, "key-auth",
		"this push's own contribution must still be included")
}
