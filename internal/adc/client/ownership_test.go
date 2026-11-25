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

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func TestSyncOwnershipSelector(t *testing.T) {
	for _, backend := range []string{"apisix", "api7ee", "custom", "apisix-standalone"} {
		for _, explicit := range []bool{false, true} {
			for _, operation := range []string{"full", "resource", "cleanup", "global-rule"} {
				name := backend + "/default/" + operation
				if explicit {
					name = backend + "/explicit/" + operation
				}
				t.Run(name, func(t *testing.T) {
					received := make(chan ADCServerRequest, 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var req ADCServerRequest
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							t.Error(err)
						}
						received <- req
						_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
					}))
					defer server.Close()
					t.Setenv("ADC_SERVER_URL", server.URL)
					c, err := New(logr.Discard(), backend, time.Second)
					require.NoError(t, err)
					config := adctypes.Config{Name: "proxy", ServerAddrs: []string{"http://apisix:9180"}}
					if explicit {
						config.BackendType = backend
					}
					owner := types.NamespacedNameKind{Kind: "Ingress", Namespace: "default", Name: "route"}
					proxy := types.NamespacedNameKind{Kind: "GatewayProxy", Namespace: "default", Name: "proxy"}
					task := Task{
						Key:       owner,
						Configs:   map[types.NamespacedNameKind]adctypes.Config{proxy: config},
						Resources: &adctypes.Resources{},
					}
					expected := map[string]string{}
					if backend != "apisix-standalone" {
						expected[label.LabelManagedBy] = "apisix-ingress-controller"
					}
					switch operation {
					case "full":
						require.NoError(t, c.UpdateConfig(context.Background(), task))
						_, err = c.Sync(context.Background())
					case "cleanup":
						require.NoError(t, c.UpdateConfig(context.Background(), task))
						err = c.Delete(context.Background(), Task{Key: owner})
					default:
						task.Labels = map[string]string{label.LabelResourceKey: "Ingress/default/route", label.LabelNamespace: "default"}
						if operation == "global-rule" {
							task.ResourceTypes = []string{"global_rule"}
						} else {
							task.ResourceTypes = []string{"service"}
						}
						expected[label.LabelResourceKey] = "Ingress/default/route"
						require.NoError(t, c.UpdateConfig(context.Background(), task))
						err = c.sync(context.Background(), task)
						assert.Equal(t, map[string]string{label.LabelResourceKey: "Ingress/default/route", label.LabelNamespace: "default"}, task.Labels)
					}
					require.NoError(t, err)
					select {
					case req := <-received:
						assert.Equal(t, backend, req.Task.Opts.Backend)
						if len(expected) == 0 {
							assert.Empty(t, req.Task.Opts.LabelSelector)
						} else {
							assert.Equal(t, expected, req.Task.Opts.LabelSelector)
						}
					default:
						t.Fatal("sync did not reach ADC")
					}
				})
			}
		}
	}
}
