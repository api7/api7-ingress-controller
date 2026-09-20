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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func listener(name string, port gatewayv1.PortNumber, hostname string) gatewayv1.Listener {
	l := gatewayv1.Listener{
		Name: gatewayv1.SectionName(name),
		Port: port,
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

// The Gateway API conformance case TLSRouteHostnameIntersection turns on this:
// four Gateways whose TLS listeners carry different hostnames all resolve to one
// physical stream listen, so a TLSRoute keeping its own hostname verbatim serves
// SNIs its listener never accepted and steals them from the route whose listener
// did.
func TestFilterTLSRouteHostnames(t *testing.T) {
	exact := gatewayv1.Hostname("abc.example.com")
	moreSpecificWildcard := gatewayv1.Hostname("*.example.com")
	lessSpecificWildcard := gatewayv1.Hostname("*.com")

	tlsListener := func(name string, hostname *gatewayv1.Hostname) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     gatewayv1.SectionName(name),
			Protocol: gatewayv1.TLSProtocolType,
			Port:     443,
			Hostname: hostname,
		}
	}
	route := func(hostnames ...gatewayv1.Hostname) *gatewayv1.TLSRoute {
		return &gatewayv1.TLSRoute{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec:       gatewayv1.TLSRouteSpec{Hostnames: hostnames},
		}
	}

	for _, tc := range []struct {
		name      string
		listener  *gatewayv1.Hostname
		hostnames []gatewayv1.Hostname
		want      []gatewayv1.Hostname
		wantErr   bool
	}{
		{
			name:      "a wildcard route narrows to the exact listener hostname",
			listener:  &exact,
			hostnames: []gatewayv1.Hostname{moreSpecificWildcard},
			want:      []gatewayv1.Hostname{exact},
		},
		{
			name:      "a broader wildcard route narrows to the listener wildcard",
			listener:  &moreSpecificWildcard,
			hostnames: []gatewayv1.Hostname{lessSpecificWildcard},
			want:      []gatewayv1.Hostname{moreSpecificWildcard},
		},
		{
			name:      "an exact route under a listener wildcard keeps its own hostname",
			listener:  &moreSpecificWildcard,
			hostnames: []gatewayv1.Hostname{exact},
			want:      []gatewayv1.Hostname{exact},
		},
		{
			name:      "a listener without a hostname leaves the route alone",
			listener:  nil,
			hostnames: []gatewayv1.Hostname{lessSpecificWildcard},
			want:      []gatewayv1.Hostname{lessSpecificWildcard},
		},
		{
			name:      "a route without hostnames takes the listener hostname",
			listener:  &moreSpecificWildcard,
			hostnames: nil,
			want:      []gatewayv1.Hostname{moreSpecificWildcard},
		},
		{
			name:      "a route without hostnames under a hostname-less listener matches anything",
			listener:  nil,
			hostnames: nil,
			want:      nil,
		},
		{
			name:      "no intersection is rejected",
			listener:  &exact,
			hostnames: []gatewayv1.Hostname{gatewayv1.Hostname("other.example.net")},
			wantErr:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gateways := []RouteParentRefContext{
				{Listeners: []gatewayv1.Listener{tlsListener("tls", tc.listener)}},
			}

			filtered, err := filterTLSRouteHostnames(gateways, route(tc.hostnames...).DeepCopy())
			if tc.wantErr {
				require.ErrorIs(t, err, ErrNoMatchingListenerHostname)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, filtered.Spec.Hostnames)
		})
	}
}
