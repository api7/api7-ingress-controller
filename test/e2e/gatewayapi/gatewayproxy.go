package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = FDescribe("Test GatewayProxy", func() {
	s := scaffold.NewDefaultScaffold()

	var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var gatewayWithProxy = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: api7
spec:
  gatewayClassName: %s
  listeners:
    - name: http
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: gateway.apisix.io
      kind: GatewayProxy
      name: api7-proxy-config
`

	var gatewayWithoutProxy = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: api7-no-proxy
spec:
  gatewayClassName: %s
  listeners:
    - name: http
      protocol: HTTP
      port: 80
`

	var gatewayProxyWithEnabledPlugin = `
apiVersion: gateway.apisix.io/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config
spec:
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        X-Proxy-Test: "enabled"
`

	var gatewayProxyWithDisabledPlugin = `
apiVersion: gateway.apisix.io/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config
spec:
  plugins:
  - name: response-rewrite
    enabled: false
    config: 
      headers:
        X-Proxy-Test: "disabled"
`

	var httpRouteForTest = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: test-route
spec:
  parentRefs:
  - name: %s
  hostnames:
  - example.com
  rules:
  - matches:
    - path:
        type: Exact
        value: /test
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	BeforeEach(func() {
		By("Create GatewayClass")
		gatewayClassName := fmt.Sprintf("api7-%d", time.Now().Unix())
		err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(3 * time.Second)

		By("Check GatewayClass condition")
		gcYaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcYaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcYaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

		By("Create Gateway with GatewayProxy")
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayWithProxy, gatewayClassName))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway with GatewayProxy")
		time.Sleep(3 * time.Second)

		By("Create Gateway without GatewayProxy")
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayWithoutProxy, gatewayClassName))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway without GatewayProxy")
		time.Sleep(3 * time.Second)
	})

	Context("Test Gateway with enabled GatewayProxy plugin", func() {
		It("Should apply plugin configuration when enabled", func() {
			By("Create GatewayProxy with enabled plugin")
			err := s.CreateResourceFromString(gatewayProxyWithEnabledPlugin)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with enabled plugin")
			time.Sleep(5 * time.Second)

			By("Create HTTPRoute for Gateway with GatewayProxy")
			err = s.CreateResourceFromString(fmt.Sprintf(httpRouteForTest, "api7"))
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute for Gateway with GatewayProxy")
			time.Sleep(5 * time.Second)

			By("Check if the plugin is applied")
			resp := s.NewAPISIXClient().
				GET("/test").
				WithHost("example.com").
				Expect().
				Status(200)

			resp.Header("X-Proxy-Test").IsEqual("enabled")
		})
	})

	Context("Test Gateway with disabled GatewayProxy plugin", func() {
		It("Should not apply plugin configuration when disabled", func() {
			By("Update GatewayProxy with disabled plugin")
			err := s.CreateResourceFromString(gatewayProxyWithDisabledPlugin)
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy with disabled plugin")
			time.Sleep(5 * time.Second)

			By("Check if the plugin is not applied")
			resp := s.NewAPISIXClient().
				GET("/test").
				WithHost("example.com").
				Expect().
				Status(200)

			resp.Header("X-Proxy-Test").IsEmpty()
		})
	})

	Context("Test Gateway without GatewayProxy", func() {
		It("Should work normally without GatewayProxy", func() {
			By("Create HTTPRoute for Gateway without GatewayProxy")
			err := s.CreateResourceFromString(fmt.Sprintf(httpRouteForTest, "api7-no-proxy"))
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute for Gateway without GatewayProxy")
			time.Sleep(5 * time.Second)

			By("Check if the route works without plugin")
			resp := s.NewAPISIXClient().
				GET("/test").
				WithHost("example.com").
				Expect().
				Status(200)

			resp.Header("X-Proxy-Test").IsEmpty()
		})
	})

	AfterEach(func() {
		By("Clean up resources")
		_ = s.DeleteResourceFromString(gatewayProxyWithEnabledPlugin)
		_ = s.DeleteResourceFromString(fmt.Sprintf(httpRouteForTest, "api7"))
		_ = s.DeleteResourceFromString(fmt.Sprintf(httpRouteForTest, "api7-no-proxy"))
		_ = s.DeleteResourceFromString(fmt.Sprintf(gatewayWithProxy, "dummy"))
		_ = s.DeleteResourceFromString(fmt.Sprintf(gatewayWithoutProxy, "dummy"))
		time.Sleep(3 * time.Second)
	})
})
