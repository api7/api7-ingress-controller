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
	"testing"

	"sigs.k8s.io/gateway-api/conformance"
	conformancev1 "sigs.k8s.io/gateway-api/conformance/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/tests"
)

var skippedTestsForSSL = []string{
	// Reason: https://github.com/kubernetes-sigs/gateway-api/blob/5c5fc388829d24e8071071b01e8313ada8f15d9f/conformance/utils/suite/suite.go#L358.  SAN includes '*'
	tests.HTTPRouteHTTPSListener.ShortName,
	tests.HTTPRouteRedirectPortAndScheme.ShortName,
}

// The API7 gateway's stream_route schema carries `sni` only, under
// additionalProperties = false, and has no tls_passthrough - it predates
// apache/apisix#13912. A stream route carrying either `snis` or
// `tls_passthrough` is rejected outright, so a TLSRoute needs its own gateway
// support before these can run here. The APISIX provider runs them already; see
// test/conformance/conformance_test.go.
var skippedTestsForGatewaySchema = []string{
	// Pinned to mode: Passthrough, which the gateway cannot serve.
	tests.TLSRouteSimpleSameNamespace.ShortName,
	tests.TLSRouteInvalidBackendRefNonexistent.ShortName,
	tests.TLSRouteInvalidBackendRefUnknownKind.ShortName,
	// Passthrough as well, and its routes carry several hostnames, which the
	// translator emits as `snis`.
	tests.TLSRouteHostnameIntersection.ShortName,
}

// TODO: HTTPRoute hostname intersection and listener hostname matching

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForGatewaySchema...)
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
		Contact:      []string{"https://github.com/apache/apisix-ingress-controller/issues"},
	}

	conformance.RunConformanceWithOptions(t, opts)
}
