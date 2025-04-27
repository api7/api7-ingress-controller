// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package scaffold

import (
	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	. "github.com/onsi/gomega"
)

func (s *Scaffold) deployDataplane() {
	svc := s.DeployGateway(framework.DataPlaneDeployOptions{
		GatewayGroupID:         s.gatewaygroupid,
		Namespace:              s.namespace,
		Name:                   "api7ee3-apisix-gateway-mtls",
		DPManagerEndpoint:      framework.DPManagerTLSEndpoint,
		SetEnv:                 true,
		SSLKey:                 framework.TestKey,
		SSLCert:                framework.TestCert,
		TLSEnabled:             true,
		ForIngressGatewayGroup: true,
		ServiceHTTPPort:        9080,
		ServiceHTTPSPort:       9443,
	})

	s.dataplaneService = svc

	err := s.newAPISIXTunnels()
	Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
}

func (s *Scaffold) newAPISIXTunnels() error {
	serviceName := "api7ee3-apisix-gateway-mtls"
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(s.dataplaneService, s.kubectlOptions, serviceName)
	if err != nil {
		return err
	}

	s.apisixHttpTunnel = httpTunnel
	s.apisixHttpsTunnel = httpsTunnel
	return nil
}
