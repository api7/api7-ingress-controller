// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apisix

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

type Headers map[string]string

var _ = Describe("Test ApisixConsumer", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Test KeyAuth", func() {
		const keyAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: test-key
`
		const defaultApisixRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /get
      - /headers
      - /anything
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: keyAuth
`
		request := func(path string, headers Headers) int {
			return s.NewAPISIXClient().GET(path).WithHeaders(headers).WithHost("httpbin").Expect().Raw().StatusCode
		}

		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, keyAuth)

			By("verify ApisixRoute with ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "invalid-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			Eventually(request).WithArguments("/get", Headers{
				"apikey": "test-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Delete ApisixConsumer")
			err := s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "test-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers", Headers{}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		PIt("SecretRef tests", func() {
		})
	})

	Context("Test BasicAuth", func() {
		const basicAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: apisix
  authParameter:
    basicAuth:
      value:
        username: test-user
        password: test-password
`
		const defaultApisixRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /get
      - /headers
      - /anything
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: basicAuth
`

		request := func(path string, username, password string) int {
			return s.NewAPISIXClient().GET(path).WithBasicAuth(username, password).WithHost("httpbin").Expect().Raw().StatusCode
		}
		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, basicAuth)

			By("verify ApisixRoute with ApisixConsumer")
			Eventually(request).WithArguments("/get", "invalid-username", "invalid-password").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			Eventually(request).WithArguments("/get", "test-user", "test-password").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Delete ApisixConsumer")
			err := s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			Eventually(request).WithArguments("/get", "test-user", "test-password").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers", "", "").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		PIt("SecretRef tests", func() {
		})
	})
})
