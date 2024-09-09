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
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
)

func (s *Scaffold) deployDataplane() {
	svc := s.DeployGateway(framework.DataPlaneDeployOptions{
		Namespace:              s.namespace,
		Name:                   "api7ee3-apisix-gateway-mtls",
		DPManagerEndpoint:      framework.DPManagerTLSEndpoint,
		SetEnv:                 true,
		SSLKey:                 framework.TestKey,
		SSLCert:                framework.TestCert,
		TLSEnabled:             true,
		ForIngressGatewayGroup: true,
	})

	s.dataplaneService = svc

	err := s.newAPISIXTunnels()
	Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
}

func (s *Scaffold) newAPISIXTunnels() error {
	var (
		httpNodePort  int
		httpsNodePort int
		httpPort      int
		httpsPort     int
		serviceName   = "api7ee3-apisix-gateway-mtls"
	)

	svc := s.dataplaneService
	for _, port := range svc.Spec.Ports {
		if port.Name == "http" {
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		} else if port.Name == "https" {
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		}
	}
	s.apisixHttpTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, serviceName,
		httpNodePort, httpPort)
	s.apisixHttpsTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, serviceName,
		httpsNodePort, httpsPort)

	if err := s.apisixHttpTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixHttpTunnel.Close)
	if err := s.apisixHttpsTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixHttpsTunnel.Close)
	return nil
}
