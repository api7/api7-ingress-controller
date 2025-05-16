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

package gatewayapi

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/retry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test HTTPRoute", func() {
	s := scaffold.NewDefaultScaffold()

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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

	var gatewayClassYaml = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`
	var defaultGatewayHTTPS = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTPS
      port: 443
      hostname: api6.com
      tls:
        certificateRefs:
        - kind: Secret
          group: ""
          name: test-apisix-tls
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`
	var ApplyHTTPRoute = func(nn types.NamespacedName, spec string) {
		err := s.CreateResourceFromString(spec)
		Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute %s", nn)
		framework.HTTPRouteMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
			types.NamespacedName{},
			types.NamespacedName{Namespace: cmp.Or(nn.Namespace, s.Namespace()), Name: nn.Name},
			metav1.Condition{
				Type:   string(gatewayv1.RouteConditionAccepted),
				Status: metav1.ConditionTrue,
			},
		)
	}
	var ApplyHTTPRoutePolicy = func(refNN, hrpNN types.NamespacedName, spec string) {
		err := s.CreateResourceFromString(spec)
		Expect(err).NotTo(HaveOccurred(), "creating HTTPRoutePolicy %s", hrpNN)
		framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second, refNN, hrpNN, metav1.Condition{
			Type:   string(v1alpha2.PolicyConditionAccepted),
			Status: metav1.ConditionTrue,
		})
	}

	var beforeEachHTTP = func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())
		err := s.CreateResourceFromString(gatewayProxy)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("check GatewayClass condition")
		framework.GatewayClassMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
			types.NamespacedName{Namespace: s.Namespace(), Name: gatewayClassName},
			metav1.Condition{
				Type:    string(gatewayv1.GatewayClassConditionStatusAccepted),
				Status:  metav1.ConditionTrue,
				Message: "the gatewayclass has been accepted by the apisix-ingress-controller",
			},
		)

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		By("check Gateway condition")
		framework.GatewayMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
			types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
			metav1.Condition{
				Type:    string(gatewayv1.GatewayConditionAccepted),
				Status:  metav1.ConditionTrue,
				Message: "the gateway has been accepted by the apisix-ingress-controller",
			},
		)
	}

	var beforeEachHTTPS = func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())
		err := s.CreateResourceFromString(gatewayProxy)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		secretName := _secretName
		createSecret(s, secretName)

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("check GatewayClass condition")
		framework.GatewayClassMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
			types.NamespacedName{Namespace: s.Namespace(), Name: gatewayClassName},
			metav1.Condition{
				Type:    string(gatewayv1.GatewayClassConditionStatusAccepted),
				Status:  metav1.ConditionTrue,
				Message: "the gatewayclass has been accepted by the apisix-ingress-controller",
			},
		)

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayHTTPS, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		By("check Gateway condition")
		framework.GatewayMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
			types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
			metav1.Condition{
				Type:    string(gatewayv1.GatewayConditionAccepted),
				Status:  metav1.ConditionTrue,
				Message: "the gateway has been accepted by the apisix-ingress-controller",
			},
		)
	}
	Context("HTTPRoute with HTTPS Gateway", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - api6.com
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(beforeEachHTTPS)

		It("Create/Update/Delete HTTPRoute", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, exactRouteByGet)

			By("access data plane to check the HTTPRoute")
			request := func() int {
				return s.NewAPISIXHttpsClient("api6.com").
					GET("/get").
					WithHost("api6.com").
					Expect().
					Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(exactRouteByGet)
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})
	})

	Context("HTTPRoute with Multiple Gateway", func() {
		var additionalGatewayGroupID string
		var additionalNamespace string
		var additionalGatewayClassName string

		var additionalGatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: additional-proxy-config
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

		var additionalGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: additional-gateway
spec:
  gatewayClassName: %s
  listeners:
    - name: http-additional
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: additional-proxy-config
`

		// HTTPRoute that references both gateways
		var multiGatewayHTTPRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: multi-gateway-route
spec:
  parentRefs:
  - name: apisix
    namespace: %s
  - name: additional-gateway
    namespace: %s
  hostnames:
  - httpbin.example
  - httpbin-additional.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(func() {
			beforeEachHTTP()

			By("Create additional gateway group")
			var err error
			additionalGatewayGroupID, additionalNamespace, err = s.CreateAdditionalGatewayGroup("multi-gw")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			By("Create additional GatewayProxy")
			// Get admin key for the additional gateway group
			resources, exists := s.GetAdditionalGatewayGroup(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			By("Create additional GatewayClass")
			additionalGatewayClassName = fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, additionalGatewayClassName, s.GetControllerName()), "")
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayClass")

			By("Check additional GatewayClass condition")
			framework.GatewayClassMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
				types.NamespacedName{Namespace: s.Namespace(), Name: additionalGatewayClassName},
				metav1.Condition{
					Type:    string(gatewayv1.GatewayClassConditionStatusAccepted),
					Status:  metav1.ConditionTrue,
					Message: "the gatewayclass has been accepted by the apisix-ingress-controller",
				},
			)

			additionalGatewayProxy := fmt.Sprintf(additionalGatewayProxyYaml, framework.DashboardTLSEndpoint, resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(additionalGatewayProxy, additionalNamespace)
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayProxy")

			By("Create additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(
				fmt.Sprintf(additionalGateway, additionalGatewayClassName),
				additionalNamespace,
			)
			Expect(err).NotTo(HaveOccurred(), "creating additional Gateway")
			time.Sleep(5 * time.Second)
		})

		It("HTTPRoute should be accessible through both gateways", func() {
			request := func(client *httpexpect.Expect, host string) int {
				return client.GET("/get").WithHost(host).Expect().Raw().StatusCode
			}

			By("Create HTTPRoute referencing both gateways")
			multiGatewayRoute := fmt.Sprintf(multiGatewayHTTPRoute, s.Namespace(), additionalNamespace)
			ApplyHTTPRoute(types.NamespacedName{Name: "multi-gateway-route"}, multiGatewayRoute)

			By("Access through default gateway")
			Eventually(request).WithArguments(s.NewAPISIXClient(), "httpbin.example").
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Access through additional gateway")
			client, err := s.NewAPISIXClientForGatewayGroup(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")
			Eventually(request).WithArguments(client, "httpbin-additional.example").
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Delete Additional Gateway")
			err = s.DeleteResourceFromStringWithNamespace(fmt.Sprintf(additionalGateway, additionalGatewayClassName), additionalNamespace)
			Expect(err).NotTo(HaveOccurred(), "deleting additional Gateway")

			By("HTTPRoute should still be accessible through default gateway")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusOK)

			By("HTTPRoute should not be accessible through additional gateway")
			client, err = s.NewAPISIXClientForGatewayGroup(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")
			Eventually(request).WithArguments(client, "httpbin-additional.example").
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})
	})

	Context("HTTPRoute Base", func() {
		var httprouteWithExternalName = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: postman-echo.com
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.external
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-external-domain
      port: 80
`
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var exactRouteByGet2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin2
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin2.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var invalidBackendPort = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-multiple-port
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
    - name: invalid
      port: 10031
      protocol: TCP
      targetPort: 10031
    - name: http2
      port: 8080
      protocol: TCP
      targetPort: 80
  type: ClusterIP
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-multiple-port
      port: 80
`

		BeforeEach(beforeEachHTTP)

		It("Create/Update/Delete HTTPRoute", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, exactRouteByGet)

			request := func(host string) int {
				if host == "" {
					return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
				}
				return s.NewAPISIXClient().GET("/get").WithHost(host).Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithArguments("httpbin.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().
				GET("/get").
				Expect().
				Status(404)

			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(exactRouteByGet)
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			Eventually(request).WithArguments("httpbin.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("Delete Gateway after apply HTTPRoute", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, exactRouteByGet)

			request := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin.example").Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("delete Gateway")
			err := s.DeleteResource("Gateway", "apisix")
			Expect(err).NotTo(HaveOccurred(), "deleting Gateway")
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("Proxy External Service", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, httprouteWithExternalName)

			By("checking the external service response")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin.external").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("Match Port", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, invalidBackendPort)

			serviceResources, err := s.DefaultDataplaneResource().Service().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing services")
			Expect(serviceResources).To(HaveLen(1), "checking service length")

			serviceResource := serviceResources[0]
			nodes := serviceResource.Upstream.Nodes
			Expect(nodes).To(HaveLen(1), "checking nodes length")
			Expect(nodes[0].Port).To(Equal(80))
		})

		It("Delete HTTPRoute during restart", func() {
			By("create HTTPRoute httpbin and httpbin2")
			var (
				httpbinNN  = types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}
				httpbin2NN = types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin2"}
			)
			for nn, spec := range map[types.NamespacedName]string{
				httpbinNN:  exactRouteByGet,
				httpbin2NN: exactRouteByGet2,
			} {
				ApplyHTTPRoute(nn, spec)
			}

			request := func(host string) int {
				return s.NewAPISIXClient().GET("/get").WithHost(host).Expect().Raw().StatusCode
			}
			Eventually(request).WithArguments("httpbin.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments("httpbin2.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			s.ScaleIngress(0)

			By("delete HTTPRoute httpbin2")
			err := s.DeleteResource("HTTPRoute", "httpbin2")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute httpbin2")

			s.ScaleIngress(1)
			Eventually(request).WithArguments("httpbin.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments("httpbin2.example").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})
	})

	Context("HTTPRoute Rule Match", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var varsRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
      headers:
        - type: Exact
          name: X-Route-Name
          value: httpbin
    # name: get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		const httpRoutePolicy = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
    # sectionName: get
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin-1
    sectionName: get
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
  - - arg_hrp_name
    - ==
    - http-route-policy-0
`

		var prefixRouteByStatus = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /status
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var methodRouteGETAndDELETEByAnything = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /anything
      method: GET
    - path:
        type: Exact
        value: /anything
      method: DELETE
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		BeforeEach(beforeEachHTTP)

		It("HTTPRoute Exact Match", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, exactRouteByGet)

			request := func(uri string) int {
				return s.NewAPISIXClient().GET(uri).WithHost("httpbin.example").Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithArguments("/get").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments("/get/xxx").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("HTTPRoute Prefix Match", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, prefixRouteByStatus)

			request := func(uri string) int {
				return s.NewAPISIXClient().GET(uri).WithHost("httpbin.example").Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithArguments("/status/200").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments("/status/201").WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusCreated))
		})

		It("HTTPRoute Method Match", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, methodRouteGETAndDELETEByAnything)

			request := func(method string) int {
				return s.NewAPISIXClient().Request(method, "/anything").WithHost("httpbin.example").Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithArguments(http.MethodGet).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments(http.MethodDelete).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			Eventually(request).WithArguments(http.MethodPost).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("HTTPRoute Vars Match", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)

			request := func(headers map[string]string) int {
				cli := s.NewAPISIXClient().GET("/get").WithHost("httpbin.example")
				for k, v := range headers {
					cli = cli.WithHeader(k, v)
				}
				return cli.Expect().Raw().StatusCode
			}
			By("access dataplane to check the HTTPRoute")
			Eventually(request).WithArguments(map[string]string{}).
				WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
			Eventually(request).WithArguments(map[string]string{"X-Route-Name": "httpbin"}).
				WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("HTTPRoutePolicy in effect", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithHost("httpbin.example").WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("create HTTPRoutePolicy")
			ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "http-route-policy-0").
				WithQuery("hrp_name", "http-route-policy-0").
				Expect().
				Status(http.StatusOK)

			By("update HTTPRoutePolicy")
			const changedHTTPRoutePolicy = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
    # sectionName: get
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - new-hrp-name
`
			ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				changedHTTPRoutePolicy,
			)

			// use the old vars cannot match any route
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					WithHeader("X-HRP-Name", "http-route-policy-0").
					WithQuery("hrp_name", "http-route-policy-0").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			// use the new vars can match the route
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "new-hrp-name").
				Expect().
				Status(http.StatusOK)

			By("delete the HTTPRoutePolicy")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-0")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy")
			Eventually(func() string {
				_, err := s.GetResourceYaml("HTTPRoutePolicy", "http-route-policy-0")
				return err.Error()
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`httproutepolicies.apisix.apache.org "http-route-policy-0" not found`))
			// access the route without additional vars should be OK
			message := retry.DoWithRetry(s.GinkgoT, "", 10, time.Second, func() (string, error) {
				statusCode := s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
				if statusCode != http.StatusOK {
					return "", errors.Errorf("unexpected status code: %v", statusCode)
				}
				return "request OK", nil
			})
			s.Logf(message)
		})

		It("HTTPRoutePolicy conflicts", func() {
			const httpRoutePolicy0 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy1 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-1
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy1Priority20 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-1
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 20
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy2 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-2
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin-1
  priority: 30
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)

			By("create HTTPRoutePolices")
			for name, spec := range map[string]string{
				"http-route-policy-0": httpRoutePolicy0,
				"http-route-policy-1": httpRoutePolicy1,
				"http-route-policy-2": httpRoutePolicy2,
			} {
				ApplyHTTPRoutePolicy(
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					spec,
				)
			}
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1", "http-route-policy-2"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}

			// assert that conflict policies are not in effect
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("delete HTTPRoutePolicies")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-2")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy %s", "http-route-policy-2")
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionTrue,
						Reason: string(v1alpha2.PolicyReasonAccepted),
					},
				)
			}
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("update HTTPRoutePolicy")
			ApplyHTTPRoutePolicy(
				types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-1"},
				httpRoutePolicy1Priority20,
			)
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("HTTPRoutePolicy status changes on HTTPRoute deleting", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)

			By("create HTTPRoutePolicy")
			ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				Expect().
				Status(http.StatusNotFound)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "http-route-policy-0").
				WithQuery("hrp_name", "http-route-policy-0").
				Expect().
				Status(http.StatusOK)

			By("delete the HTTPRoute, assert the HTTPRoutePolicy's status will be changed")
			err := s.DeleteResource("HTTPRoute", "httpbin")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			message := retry.DoWithRetry(s.GinkgoT, "request the deleted route", 10, time.Second, func() (string, error) {
				statusCode := s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					WithHeader("X-HRP-Name", "http-route-policy-0").
					WithQuery("hrp_name", "http-route-policy-0").
					Expect().Raw().StatusCode
				if statusCode != http.StatusNotFound {
					return "", errors.Errorf("unexpected status code: %v", statusCode)
				}
				return "the route is deleted", nil
			})
			s.Logf(message)

			err = framework.PollUntilHTTPRoutePolicyHaveStatus(s.K8sClient, 8*time.Second, types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				func(_ v1alpha1.HTTPRoutePolicy, status v1alpha1.PolicyStatus) bool {
					return len(status.Ancestors) == 0
				},
			)
			Expect(err).NotTo(HaveOccurred(), "HTPRoutePolicy.Status should has no ancestor")
		})
	})

	Context("HTTPRoute Filters", func() {
		var reqHeaderModifyByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestHeaderModifier
      requestHeaderModifier:
        add:
        - name: X-Req-Add
          value: "add"
        set:
        - name: X-Req-Set
          value: "set"
        remove:
        - X-Req-Removed
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var respHeaderModifyByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        add:
        - name: X-Resp-Add
          value: "add"
        set:
        - name: X-Resp-Set
          value: "set"
        remove:
        - Server
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var httpsRedirectByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestRedirect
      requestRedirect:
        scheme: https
        port: 9443
`

		var hostnameRedirectByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestRedirect
      requestRedirect:
        hostname: httpbin.org
        statusCode: 301
`

		var replacePrefixMatch = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /replace
    filters:
    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplacePrefixMatch
          replacePrefixMatch: /status
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var replaceFullPathAndHost = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /replace
    filters:
    - type: URLRewrite
      urlRewrite:
        hostname: replace.example.org
        path:
          type: ReplaceFullPath
          replaceFullPath: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var echoPlugin = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: example-plugin-config
spec:
  plugins:
  - name: echo
    config:
      body: "Hello, World!!"
`
		var echoPluginUpdated = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: example-plugin-config
spec:
  plugins:
  - name: echo
    config:
      body: "Updated"
`
		var extensionRefEchoPlugin = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: example-plugin-config
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(beforeEachHTTP)

		It("HTTPRoute RequestHeaderModifier", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, reqHeaderModifyByHeaders)

			By("access dataplane to check the HTTPRoute")
			respExp := s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				WithHeader("X-Req-Add", "test").
				WithHeader("X-Req-Removed", "test").
				WithHeader("X-Req-Set", "test").
				Expect()

			respExp.Status(200)
			respExp.Body().
				Contains(`"X-Req-Add": "test,add"`).
				Contains(`"X-Req-Set": "set"`).
				NotContains(`"X-Req-Removed": "remove"`)

		})

		It("HTTPRoute ResponseHeaderModifier", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, respHeaderModifyByHeaders)

			By("access dataplane to check the HTTPRoute")
			respExp := s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect()

			respExp.Status(200)
			respExp.Header("X-Resp-Add").IsEqual("add")
			respExp.Header("X-Resp-Set").IsEqual("set")
			respExp.Header("Server").IsEmpty()
			respExp.Body().
				NotContains(`"X-Resp-Add": "add"`).
				NotContains(`"X-Resp-Set": "set"`).
				NotContains(`"Server"`)
		})

		It("HTTPRoute RequestRedirect", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, httpsRedirectByHeaders)

			s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusFound).
				Header("Location").IsEqual("https://httpbin.example:9443/headers")

			By("update HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, hostnameRedirectByHeaders)

			s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusMovedPermanently).
				Header("Location").IsEqual("http://httpbin.org/headers")
		})

		It("HTTPRoute RequestMirror", func() {
			echoRoute := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo
spec:
  selector:
    matchLabels:
      app: echo
  replicas: 1
  template:
    metadata:
      labels:
        app: echo
    spec:
      containers:
      - name: echo
        image: jmalloc/echo-server:latest
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: echo-service
spec:
  selector:
    app: echo
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, echoRoute)
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/headers").
					WithHeader("Host", "httpbin.example").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			echoLogs := s.GetDeploymentLogs("echo")
			Expect(echoLogs).To(ContainSubstring("GET /headers"))
		})

		It("HTTPRoute URLRewrite with ReplaceFullPath And Hostname", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, replaceFullPathAndHost)

			By("/replace/201 should be rewritten to /headers")
			s.NewAPISIXClient().GET("/replace/201").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK).
				Body().
				Contains("replace.example.org")

			By("/replace/500 should be rewritten to /headers")
			s.NewAPISIXClient().GET("/replace/500").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK).
				Body().
				Contains("replace.example.org")
		})

		It("HTTPRoute URLRewrite with ReplacePrefixMatch", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, replacePrefixMatch)

			By("/replace/201 should be rewritten to /status/201")
			s.NewAPISIXClient().GET("/replace/201").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusCreated)

			By("/replace/500 should be rewritten to /status/500")
			s.NewAPISIXClient().GET("/replace/500").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusInternalServerError)
		})

		It("HTTPRoute ExtensionRef", func() {
			By("create HTTPRoute")
			err := s.CreateResourceFromString(echoPlugin)
			Expect(err).NotTo(HaveOccurred(), "creating PluginConfig")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, extensionRefEchoPlugin)

			s.NewAPISIXClient().GET("/get").
				WithHeader("Host", "httpbin.example").
				Expect().
				Body().
				Contains("Hello, World!!")

			err = s.CreateResourceFromString(echoPluginUpdated)
			Expect(err).NotTo(HaveOccurred(), "updating PluginConfig")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().GET("/get").
				WithHeader("Host", "httpbin.example").
				Expect().
				Body().
				Contains("Updated")
		})
	})

	Context("HTTPRoute Multiple Backend", func() {
		var sameWeight = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
 `
		var oneWeight = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 100
    - name: nginx
      port: 80
      weight: 0
 `

		BeforeEach(func() {
			beforeEachHTTP()
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
			})
		})
		It("HTTPRoute Canary", func() {
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, sameWeight)

			var (
				hitNginxCnt   = 0
				hitHttpbinCnt = 0
			)
			for i := 0; i < 100; i++ {
				body := s.NewAPISIXClient().GET("/get").
					WithHeader("Host", "httpbin.example").
					Expect().
					Status(http.StatusOK).
					Body().Raw()

				if strings.Contains(body, "Hello") {
					hitNginxCnt++
				} else {
					hitHttpbinCnt++
				}
			}
			Expect(hitNginxCnt - hitHttpbinCnt).To(BeNumerically("~", 0, 2))

			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, oneWeight)

			hitNginxCnt = 0
			hitHttpbinCnt = 0
			for i := 0; i < 100; i++ {
				body := s.NewAPISIXClient().GET("/get").
					WithHeader("Host", "httpbin.example").
					Expect().
					Status(http.StatusOK).
					Body().Raw()

				if strings.Contains(body, "Hello") {
					hitNginxCnt++
				} else {
					hitHttpbinCnt++
				}
			}
			Expect(hitHttpbinCnt - hitNginxCnt).To(Equal(100))
		})
	})

	Context("HTTPRoute with GatewayProxy Update", func() {
		var additionalGatewayGroupID string

		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var updatedGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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

		BeforeEach(beforeEachHTTP)

		It("Should sync HTTPRoute when GatewayProxy is updated", func() {
			By("create HTTPRoute")
			ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, exactRouteByGet)

			By("verify HTTPRoute works")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("create additional gateway group to get new admin key")
			var err error
			additionalGatewayGroupID, _, err = s.CreateAdditionalGatewayGroup("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			resources, exists := s.GetAdditionalGatewayGroup(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			client, err := s.NewAPISIXClientForGatewayGroup(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("HTTPRoute not found for additional gateway group")
			client.
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(404)

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, framework.DashboardTLSEndpoint, resources.AdminAPIKey)
			err = s.CreateResourceFromString(updatedProxy)
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("verify HTTPRoute works for additional gateway group")
			client.
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)
		})
	})

	/*
		Context("HTTPRoute Status Updated", func() {
		})

		Context("HTTPRoute ParentRefs With Multiple Gateway", func() {
		})


		Context("HTTPRoute BackendRefs Discovery", func() {
		})
	*/
})
