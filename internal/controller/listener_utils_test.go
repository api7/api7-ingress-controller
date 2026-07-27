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
	"testing"

	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func listener(name string, port int32, hostname string) gatewayv1.Listener {
	l := gatewayv1.Listener{
		Name: gatewayv1.SectionName(name),
		Port: gatewayv1.PortNumber(port),
	}
	if hostname != "" {
		h := gatewayv1.Hostname(hostname)
		l.Hostname = &h
	}
	return l
}

func TestAppendListenersKeepsSameNameOnDifferentPorts(t *testing.T) {
	// Listener names are unique per Gateway only, and this slice spans every
	// Gateway a route attaches to.
	got := appendListeners(nil,
		listener("http", 80, ""),
		listener("http", 8080, ""),
		listener("http", 80, ""),
	)

	require.Len(t, got, 2)
	require.Equal(t, gatewayv1.PortNumber(80), got[0].Port)
	require.Equal(t, gatewayv1.PortNumber(8080), got[1].Port)
}

func TestListenersForGatewayContext(t *testing.T) {
	all := []gatewayv1.Listener{
		listener("http", 80, ""),
		listener("https", 8080, "foo.example.com"),
	}
	gateway := &gatewayv1.Gateway{}
	gateway.Spec.Listeners = all

	t.Run("matched listeners win over the gateway spec", func(t *testing.T) {
		// A parentRef targeting port 8080 must not pull in the :80 listener.
		got := listenersForGatewayContext(RouteParentRefContext{
			Gateway:   gateway,
			Listeners: []gatewayv1.Listener{all[1]},
		})
		require.Equal(t, []gatewayv1.Listener{all[1]}, got)
	})

	t.Run("falls back to the single matched listener", func(t *testing.T) {
		got := listenersForGatewayContext(RouteParentRefContext{
			Gateway:  gateway,
			Listener: &all[0],
		})
		require.Equal(t, []gatewayv1.Listener{all[0]}, got)
	})

	t.Run("falls back to sectionName lookup", func(t *testing.T) {
		got := listenersForGatewayContext(RouteParentRefContext{
			Gateway:      gateway,
			ListenerName: "https",
		})
		require.Equal(t, []gatewayv1.Listener{all[1]}, got)
	})

	t.Run("falls back to every listener when nothing matched", func(t *testing.T) {
		got := listenersForGatewayContext(RouteParentRefContext{Gateway: gateway})
		require.Equal(t, all, got)
	})
}

// TestGetMinimumHostnameIntersectionUsesMatchedListeners pins the intersection to
// the listeners the parentRef actually selected. A parentRef selecting by port
// leaves ListenerName empty, so filtering by name alone would let a route keep a
// hostname that only a listener on a different port serves.
func TestGetMinimumHostnameIntersectionUsesMatchedListeners(t *testing.T) {
	gateway := &gatewayv1.Gateway{}
	gateway.Spec.Listeners = []gatewayv1.Listener{
		listener("http-a", 80, "a.example.com"),
		listener("http-b", 8080, "b.example.com"),
	}
	portSelected := []RouteParentRefContext{{
		Gateway:   gateway,
		Listeners: []gatewayv1.Listener{gateway.Spec.Listeners[0]},
	}}

	require.Equal(t, gatewayv1.Hostname("a.example.com"),
		getMinimumHostnameIntersection(portSelected, "a.example.com"))
	require.Empty(t, getMinimumHostnameIntersection(portSelected, "b.example.com"),
		"a hostname served only by the untargeted listener must not survive")

	// Without a matched listener the whole spec is still considered, so both
	// hostnames intersect as before.
	whole := []RouteParentRefContext{{Gateway: gateway}}
	require.Equal(t, gatewayv1.Hostname("b.example.com"),
		getMinimumHostnameIntersection(whole, "b.example.com"))
}
