package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test BackendTrafficPolicy base on HTTPRoute", func() {
	s := scaffold.NewDefaultScaffold()

	var defaultGatewayProxy = `
apiVersion: gateway.apisix.io/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config
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

	var defaultGatewayClass = `
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
  name: api7ee
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: gateway.apisix.io
      kind: GatewayProxy
      name: api7-proxy-config
`

	var defaultHTTPRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: api7ee
  hostnames:
  - "httpbin.org"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
	Context("Rewrite Upstream Host", func() {
		var createUpstreamHost = `
apiVersion: gateway.apisix.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: gateway.apisix.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayProxy, defaultGatewayClass, defaultGateway, defaultHTTPRoute)
		})
		It("should rewrite upstream host", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", createUpstreamHost, 1)
			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().Contains("httpbin.example.com")

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", updateUpstreamHost, 2)
			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().Contains("httpbin.update.example.com")

			err := s.DeleteResourceFromString(createUpstreamHost)
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().
				NotContains("httpbin.update.example.com").
				NotContains("httpbin.example.com")
		})
	})
})

var _ = Describe("Test BackendTrafficPolicy base on Ingress", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
	})

	var defaultGatewayProxy = `
apiVersion: gateway.apisix.io/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config
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
	var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7-default
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
  parameters:
    apiGroup: "gateway.apisix.io"
    kind: "GatewayProxy"
    name: "api7-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

	var defaultIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-default
spec:
  rules:
  - host: httpbin.org
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
	var beforeEach = func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(defaultGatewayProxy, framework.DashboardTLSEndpoint, s.AdminKey())

		By("create GatewayProxy")
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create IngressClass with GatewayProxy reference")
		err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")

		By("create Ingress with GatewayProxy IngressClass")
		err = s.CreateResourceFromString(defaultIngress)
		Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
		time.Sleep(5 * time.Second)
	}

	Context("Rewrite Upstream Host", func() {
		var createUpstreamHost = `
apiVersion: gateway.apisix.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: gateway.apisix.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(beforeEach)
		It("should rewrite upstream host", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", createUpstreamHost, 1)
			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().Contains("httpbin.example.com")

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", updateUpstreamHost, 2)
			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().Contains("httpbin.update.example.com")

			err := s.DeleteResourceFromString(createUpstreamHost)
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.org").
				Expect().
				Status(200).
				Body().
				NotContains("httpbin.update.example.com").
				NotContains("httpbin.example.com")
		})
	})
})
