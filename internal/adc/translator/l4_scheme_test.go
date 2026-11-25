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
	"testing"

	"github.com/stretchr/testify/assert"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func TestSetL4UpstreamScheme(t *testing.T) {
	tests := []struct {
		name    string
		current string
		scheme  string
		want    string
	}{
		{name: "unset uses the route scheme", scheme: apiv2.SchemeTCP, want: apiv2.SchemeTCP},
		{name: "policy default http is replaced", current: apiv2.SchemeHTTP, scheme: apiv2.SchemeUDP, want: apiv2.SchemeUDP},
		{name: "policy tls is kept", current: apiv2.SchemeTLS, scheme: apiv2.SchemeTCP, want: apiv2.SchemeTLS},
		{name: "policy udp is kept", current: apiv2.SchemeUDP, scheme: apiv2.SchemeTCP, want: apiv2.SchemeUDP},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &adctypes.Upstream{Scheme: tt.current}
			setL4UpstreamScheme(upstream, tt.scheme)
			assert.Equal(t, tt.want, upstream.Scheme)
		})
	}
}
