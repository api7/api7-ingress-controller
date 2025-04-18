package gatewayapi

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test BackendTrafficPolicy", func() {
	s := scaffold.NewDefaultScaffold()

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
  namespace: default
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
  namespace: default
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayClass, defaultGateway, defaultHTTPRoute)
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
