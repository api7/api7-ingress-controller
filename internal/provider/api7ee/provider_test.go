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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

type readyManager struct{}

func (readyManager) RegisterGVK(...readiness.GVKConfig) {}
func (readyManager) Start(context.Context) error        { return nil }
func (readyManager) IsReady() bool                      { return true }
func (readyManager) WaitReady(context.Context, time.Duration) bool {
	return true
}
func (readyManager) Done(client.Object, k8stypes.NamespacedName) {}

func TestStartConsumesNotificationsWhenPeriodicSyncDisabled(t *testing.T) {
	adc, err := adcclient.New(logr.Discard(), ProviderTypeAPI7EE, time.Second)
	require.NoError(t, err)

	provider := &api7eeProvider{
		client:  adc,
		readier: readyManager{},
		syncCh:  make(chan struct{}, 1),
		log:     logr.Discard(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- provider.Start(ctx)
	}()

	require.Eventually(t, provider.startUpSync.Load, time.Second, 10*time.Millisecond)
	provider.syncNotify()
	require.Eventually(t, func() bool {
		return len(provider.syncCh) == 0
	}, time.Second, 10*time.Millisecond)

	select {
	case err := <-done:
		require.Failf(t, "provider stopped", "Start returned before cancellation: %v", err)
	default:
	}

	cancel()
	assert.NoError(t, <-done)
}
