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

package api7

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test apisix.apache.org/v2 Status", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	s := scaffold.NewDefaultScaffold()

	Context("Test ApisixRoute Sync Status", func() {
		const (
			serviceSpec = `
apiVersion: v1
kind: Service
metadata:
  name: api7ee3-dashboard
spec:
  type: ExternalName
  externalName: %s
`
			gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
        - https://api7ee3-dashboard:7443
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`
			ar = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		)

		It("dataplane unavailable", func() {
			s.Deployer.ScaleIngress(0)
			By("create GatewayProxy")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyYaml, s.AdminKey()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")

			By("create Service with invalid host")
			err = s.CreateResourceFromString(fmt.Sprintf(serviceSpec, "invalid.host"))
			Expect(err).NotTo(HaveOccurred(), "creating Service")

			By("apply ApisixRoute")
			err = s.CreateResourceFromString(fmt.Sprintf(ar, s.Namespace(), s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute")

			s.Deployer.ScaleIngress(1)

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(5 * time.Minute).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)

			By("update service to dashboard")
			err = s.CreateResourceFromString(fmt.Sprintf(serviceSpec, "api7ee3-dashboard.api7-ee-e2e.svc.cluster.local"))
			Expect(err).NotTo(HaveOccurred(), "updating Service")

			By("check ApisixRoute status after scaling up")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(60 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "True"`),
						ContainSubstring(`reason: Accepted`),
					),
				)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
		})
	})
})
