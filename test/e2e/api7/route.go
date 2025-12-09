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
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test apisix.apache.org/v2 Status", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s = scaffold.NewScaffold(scaffold.Options{
			// for triggering the sync
			SyncPeriod: 3 * time.Second,
		})
	)

	Context("Test ApisixRoute Sync Status", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			err := s.CreateResourceFromString(s.GetGatewayProxySpec())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		const ar = `
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

		FIt("dataplane unavailable", func() {
			s.Deployer.ScaleIngress(0)
			By("apply ApisixRoute")
			err := s.CreateResourceFromString(fmt.Sprintf(ar, s.Namespace(), s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute")

			By("update service to invalid host")
			k8sservice, err := s.GetService(framework.Namespace, framework.DashboardServiceName)
			Expect(err).NotTo(HaveOccurred(), "getting service")
			oldSpec := k8sservice.Spec
			k8sservice.Spec = corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: "invalid.host",
			}
			newServiceYaml, err := yaml.Marshal(k8sservice)
			Expect(err).NotTo(HaveOccurred(), "marshalling service")
			err = s.CreateResourceFromString(string(newServiceYaml))
			Expect(err).NotTo(HaveOccurred(), "creating service")

			s.Deployer.ScaleIngress(1)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Headers: map[string]string{"Host": "httpbin"},
				Check:   scaffold.WithExpectedStatus(404),
			})

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(60 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)

			By("update service to original spec")
			k8sservice, err = s.GetService(framework.Namespace, framework.DashboardServiceName)
			k8sservice.Spec = oldSpec
			newServiceYaml, err = yaml.Marshal(k8sservice)
			Expect(err).NotTo(HaveOccurred(), "marshalling service")
			err = s.CreateResourceFromString(string(newServiceYaml))
			Expect(err).NotTo(HaveOccurred(), "creating service")

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
