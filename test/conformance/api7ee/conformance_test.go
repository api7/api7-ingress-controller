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

// The TLSRoute tests all pin their listener to mode: Passthrough, which the
// API7 gateway serves since api7/api7-ee-3-gateway#2173. One of them cannot
// pass here for the same reason it cannot on the APISIX provider, recorded in
// test/conformance/conformance_test.go: every Gateway resolves to one data
// plane address and one physical stream listen, so their SNI namespaces are
// shared and a name no listener on the addressed Gateway accepts is still
// answered by a sibling Gateway's route.
var skippedTestsForKnownGaps = []string{
	tests.TLSRouteHostnameIntersection.ShortName,
	// Terminate mode itself is covered by TLSRouteListenerTerminateSupportedKinds
	// and by the e2e TLSRoute suite. This provisional test additionally requires a
	// standalone Gateway with no GatewayProxy attached to reach Accepted=True, and
	// a stream proxy listening on the port it picks; neither holds here, so the
	// Gateway is rejected with "gateway proxy not found" before any traffic flows.
	tests.TLSRouteTerminateSimpleSameNamespace.ShortName,
}

// TODO: HTTPRoute hostname intersection and listener hostname matching

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForKnownGaps...)
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
		Contact:      []string{"https://github.com/apache/apisix-ingress-controller/issues"},
	}

	conformance.RunConformanceWithOptions(t, opts)
}
