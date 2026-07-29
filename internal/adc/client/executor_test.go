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
	"io"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

func TestHTTPADCExecutorBuildHTTPRequestCaCert(t *testing.T) {
	e := &HTTPADCExecutor{
		serverURL: "http://127.0.0.1:3000",
		log:       logr.Discard(),
	}

	build := func(config adctypes.Config) (ADCServerOpts, string) {
		req, err := e.buildHTTPRequest(context.Background(), "https://apisix:9180", config, nil, nil,
			&adctypes.Resources{}, "/sync")
		require.NoError(t, err)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var parsed ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &parsed))
		return parsed.Task.Opts, string(body)
	}

	// Without a CA bundle the request stays what an ADC server that predates caCert
	// already accepts.
	opts, raw := build(adctypes.Config{Name: "GatewayProxy/ns/name", TlsVerify: true})
	assert.Empty(t, opts.CaCert)
	assert.NotContains(t, raw, "caCert")

	const caCert = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"
	opts, raw = build(adctypes.Config{Name: "GatewayProxy/ns/name", TlsVerify: true, CaBundle: caCert})
	assert.Equal(t, caCert, opts.CaCert)
	assert.Contains(t, raw, "caCert")
	// verification stays on, otherwise the bundle would be pointless
	assert.Equal(t, false, *opts.TlsSkipVerify)
}
