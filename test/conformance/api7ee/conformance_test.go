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
	"sigs.k8s.io/gateway-api/conformance/tests"
)

var skippedTestsForSSL = []string{
	// Reason: https://github.com/kubernetes-sigs/gateway-api/blob/5c5fc388829d24e8071071b01e8313ada8f15d9f/conformance/utils/suite/suite.go#L358.  SAN includes '*'
	tests.HTTPRouteHTTPSListener.ShortName,
	tests.HTTPRouteRedirectPortAndScheme.ShortName,
}

// These fixtures create a Gateway of their own, and that Gateway carries no
// GatewayProxy, so it never leaves "gateway proxy not found" and no traffic
// flows. TLSRoute itself is exercised by the e2e suite against this provider;
// what is missing here is a way to attach the proxy to a Gateway the test
// creates, not the feature.
var skippedTestsForStandaloneGateway = []string{
	tests.TLSRouteSimpleSameNamespace.ShortName,
	tests.TLSRouteHostnameIntersection.ShortName,
	tests.TLSRouteInvalidBackendRefNonexistent.ShortName,
	tests.TLSRouteInvalidBackendRefUnknownKind.ShortName,
	tests.TLSRouteTerminateSimpleSameNamespace.ShortName,
}

// Known gaps tracked for follow-up. These are genuine gaps rather than
// architectural limits, so they are expected to shrink over time.
var skippedTestsForKnownGaps = []string{
	// The control plane rejects a second SSL that claims an SNI another one
	// already holds ("responded with status 400 Bad Request, error_msg: SNI
	// already exists"), which APISIX accepts. Adding the HTTPS listener this
	// test asks for therefore fails the sync and the Gateway never reaches
	// Accepted. The controller has to reconcile SSLs by SNI for this provider.
	tests.GatewayModifyListeners.ShortName,

	// A single HTTPRoute attached to several Gateways is not served from each
	// parent independently.
	tests.HTTPRouteMultipleGateways.ShortName,

	// A backendRef that cannot be resolved must still produce a route that
	// answers 500; today no route is generated at all, so the request 404s.
	tests.HTTPRouteNoBackendRefs.ShortName,
}

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForStandaloneGateway...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForKnownGaps...)
	// Implementation is left to the flags DefaultOptions already applied.
	// Assigning it here would override them and pin the report to a stale version.

	conformance.RunConformanceWithOptions(t, opts)
}
