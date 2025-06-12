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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GlobalRule", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/apisix-ingress-controller",
	})

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
  namespace: default
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

	var ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

	var ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
spec:
  ingressClassName: apisix
  rules:
  - host: globalrule.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

	Context("ApisixGlobalRule Basic Operations", func() {
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

			By("create Ingress")
			err = s.CreateResourceFromString(ingressYaml)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")
			time.Sleep(5 * time.Second)

			By("verify Ingress works")
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("globalrule.example.com").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
				Should(Equal(http.StatusOK))
		})

		It("Test GlobalRule with response-rewrite plugin", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-response-rewrite
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Global-Rule: "test-response-rewrite"
        X-Global-Test: "enabled"
`

			By("create ApisixGlobalRule with response-rewrite plugin")
			err := s.CreateResourceFromString(globalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-response-rewrite")
			Expect(err).NotTo(HaveOccurred(), "getting ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			By("verify global rule is applied - response should have custom headers")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEqual("test-response-rewrite")
			resp.Header("X-Global-Test").IsEqual("enabled")

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-response-rewrite")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")
			time.Sleep(5 * time.Second)

			By("verify global rule is removed - response should not have custom headers")
			resp = s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEmpty()
			resp.Header("X-Global-Test").IsEmpty()
		})

		It("Test GlobalRule update", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Update-Test: "version1"
`

			updatedGlobalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Update-Test: "version2"
        X-New-Header: "added"
`

			By("create initial ApisixGlobalRule")
			err := s.CreateResourceFromString(globalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify initial ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "getting ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			By("verify initial configuration")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Update-Test").IsEqual("version1")
			resp.Header("X-New-Header").IsEmpty()

			By("update ApisixGlobalRule")
			err = s.CreateResourceFromString(updatedGlobalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "updating ApisixGlobalRule")

			By("verify updated ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err = s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "getting updated ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))
			Expect(gryaml).To(ContainSubstring("observedGeneration: 2"))

			By("verify updated configuration")
			resp = s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Update-Test").IsEqual("version2")
			resp.Header("X-New-Header").IsEqual("added")

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")
		})
	})
})
