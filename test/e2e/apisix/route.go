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

var _ = Describe("Test ApisixRoute", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixRoute", func() {
		const apisixRouteSpec = `
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
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

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

		It("Basic tests", func() {
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("Test plugins in ApisixRoute", func() {
			const apisixRouteSpecPart0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			const apisixRouteSpecPart1 = ` 
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          X-Global-Rule: "test-response-rewrite"
          X-Global-Test: "enabled"
`
			By("apply ApisixRoute without plugins")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("apply ApisixRoute with plugins")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0+apisixRouteSpecPart1)
			time.Sleep(5 * time.Second)

			By("verify plugin works")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEqual("test-response-rewrite")
			resp.Header("X-Global-Test").IsEqual("enabled")

			By("remove plugin")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0)
			time.Sleep(5 * time.Second)

			By("verify no plugin works")
			resp = s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEmpty()
			resp.Header("X-Global-Test").IsEmpty()
		})

		It("Test ApisixRoute match by vars", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithHeader("X-Foo", "bar").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		It("Test ApisixRoute filterFunc", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
      filter_func: "function(vars)\n  local core = require ('apisix.core')\n  local body, err = core.request.get_body()\n  if not body then\n      return false\n  end\n\n  local data, err = core.json.decode(body)\n  if not data then\n      return false\n  end\n\n  if data['foo'] == 'bar' then\n      return true\n  end\n\n  return false\nend"
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithJSON(map[string]string{"foo": "bar"}).
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		PIt("Test ApisixRoute resolveGranularity", func() {
			// The `.Spec.HTTP[0].Backends[0].ResolveGranularity` can be "endpoints" or "service",
			// when set to "endpoints", the pod ips will be used; or the service ClusterIP or ExternalIP will be used when it set to "service",

			// In the current implementation, pod ips are always used.
			// So the case is pending for now.
		})

		PIt("Test ApisixRoute subset", func() {
			// route.Spec.HTTP[].Backends[].Subset depends on ApisixUpstream.
			// ApisixUpstream is not implemented yet.
			// So the case is pending for now
		})

		PIt("Test ApisixRoute reference ApisixUpstream", func() {
			// This case depends on ApisixUpstream.
			// ApisixUpstream is not implemented yet.
			// So the case is pending for now.
		})
	})
})
