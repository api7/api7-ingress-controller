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

package readiness

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var httpRouteGVK = gatewayv1.SchemeGroupVersion.WithKind("HTTPRoute")

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, gatewayv1.Install(scheme))
	return scheme
}

func newHTTPRoute(name string) *gatewayv1.HTTPRoute {
	route := &gatewayv1.HTTPRoute{}
	route.Namespace = "default"
	route.Name = name
	return route
}

// failingClient makes the initial List fail, the path on which Start returns
// early.
type failingClient struct {
	client.Client
}

func (failingClient) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return errors.New("api server unavailable")
}

// TestStartUnblocksDoneOnListFailure covers the deadlock: Done() waits on the
// started channel, so a Start() that gives up on a failed List must still close
// it. Otherwise every reconciler blocks forever in its deferred Done().
func TestStartUnblocksDoneOnListFailure(t *testing.T) {
	r := NewReadinessManager(failingClient{}, logr.Discard())
	r.RegisterGVK(GVKConfig{GVKs: []schema.GroupVersionKind{httpRouteGVK}})

	require.Error(t, r.Start(context.Background()))

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Done(newHTTPRoute("route"), k8stypes.NamespacedName{Namespace: "default", Name: "route"})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Done() blocked after Start() failed; r.started was never closed")
	}
}

// TestStartAndDoneDoNotRaceOnState covers the concurrent map access: Start()
// reads the state to decide initial readiness while Done(), released by the same
// started channel, deletes from it. Run under -race.
func TestStartAndDoneDoNotRaceOnState(t *testing.T) {
	scheme := newTestScheme(t)
	objs := []client.Object{newHTTPRoute("a"), newHTTPRoute("b")}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

	r := NewReadinessManager(cli, logr.Discard())
	r.RegisterGVK(GVKConfig{GVKs: []schema.GroupVersionKind{httpRouteGVK}})

	var wg sync.WaitGroup
	for _, name := range []string{"a", "b"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Done(newHTTPRoute(name), k8stypes.NamespacedName{Namespace: "default", Name: name})
		}()
	}

	require.NoError(t, r.Start(context.Background()))
	wg.Wait()

	assert.True(t, r.IsReady(), "every tracked resource was marked done")
	assert.True(t, r.WaitReady(context.Background(), time.Second))
}

// TestStartWithNothingToTrackIsImmediatelyReady keeps the empty-state shortcut
// working now that the started channel is closed by a defer.
func TestStartWithNothingToTrackIsImmediatelyReady(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	r := NewReadinessManager(cli, logr.Discard())
	r.RegisterGVK(GVKConfig{GVKs: []schema.GroupVersionKind{httpRouteGVK}})

	require.NoError(t, r.Start(context.Background()))
	assert.True(t, r.IsReady())
}

// TestWaitReadyReportsTimeout pins the timeout as a failure: reporting ready on
// expiry would let a full sync push incomplete data to the data plane, which is
// what the readiness gate exists to prevent.
func TestWaitReadyReportsTimeout(t *testing.T) {
	scheme := newTestScheme(t)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newHTTPRoute("a")).Build()

	r := NewReadinessManager(cli, logr.Discard())
	r.RegisterGVK(GVKConfig{GVKs: []schema.GroupVersionKind{httpRouteGVK}})
	require.NoError(t, r.Start(context.Background()))

	assert.False(t, r.WaitReady(context.Background(), 50*time.Millisecond))
}
