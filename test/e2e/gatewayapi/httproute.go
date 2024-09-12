package gatewayapi

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test HTTPRoute", func() {
	s := scaffold.NewDefaultScaffold()

	var defautlGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var defautlGateway = `
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

	var ResourceApplied = func(resourType, resourceName, resourceRaw string, observedGeneration int) {
		Expect(s.CreateResourceFromString(resourceRaw)).
			NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourType))

		Eventually(func() string {
			hryaml, err := s.GetResourceYaml(resourType, resourceName)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourType))
			return hryaml
		}, "8s", "2s").
			Should(
				SatisfyAll(
					ContainSubstring(`status: "True"`),
					ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
				),
				fmt.Sprintf("checking %s condition status", resourType),
			)
		time.Sleep(1 * time.Second)
	}
	var beforeEach = func() {
		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("api7-%d", time.Now().Unix())
		err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defautlGatewayClass, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(5 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

		By("create Gateway")
		err = s.CreateResourceFromString(fmt.Sprintf(defautlGateway, gatewayClassName))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", "api7ee")
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the api7-ingress-controller"), "checking Gateway condition message")
	}

	Context("HTTPRoute Base", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: api7ee
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
		var addMatch = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: api7ee
  hostnames:
  - httpbin.example
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

		var addRule = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: api7ee
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
  - matches: 
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(beforeEach)

		It("Create/Updtea/Delete HTTPRoute", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access gateway to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				Expect().
				Status(404)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(exactRouteByGet)
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(404)
		})

		It("HTTPRoute Updated", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access gateway to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("update HTTPRoute, add match")
			ResourceApplied("HTTPRoute", "httpbin", addMatch, 2)

			By("access gateway to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("reset the HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 3)

			By("access gateway to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect().
				Status(404)

			By("update HTTPRoute, add rule")
			ResourceApplied("HTTPRoute", "httpbin", addRule, 4)

			By("access gateway to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/ip").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect().
				Status(404)

			By("reset the HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 5)
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/ip").
				WithHost("httpbin.example").
				Expect().
				Status(404)

			s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect().
				Status(404)
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
  - name: api7ee
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

		var prefixRouteByStatus = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: api7ee
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
  - name: api7ee
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
		BeforeEach(beforeEach)

		It("HTTPRoute Exact Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get/xxx").
				WithHost("httpbin.example").
				Expect().
				Status(404)
		})

		It("HTTPRoute Prefix Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", prefixRouteByStatus, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/status/200").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/status/201").
				WithHost("httpbin.example").
				Expect().
				Status(201)
		})

		It("HTTPRoute Method Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", methodRouteGETAndDELETEByAnything, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				DELETE("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				POST("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(404)
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
  - name: api7ee
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
  - name: api7ee
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
  - name: api7ee
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
  - name: api7ee
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

		BeforeEach(beforeEach)

		It("HTTPRoute RequestHeaderModifier", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", reqHeaderModifyByHeaders, 1)

			By("access daataplane to check the HTTPRoute")
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
			ResourceApplied("HTTPRoute", "httpbin", respHeaderModifyByHeaders, 1)

			By("access daataplane to check the HTTPRoute")
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
			ResourceApplied("HTTPRoute", "httpbin", httpsRedirectByHeaders, 1)

			s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusFound).
				Header("Location").IsEqual("https://httpbin.example:9443/headers")

			By("update HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", hostnameRedirectByHeaders, 2)

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
  - name: api7ee
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
			ResourceApplied("HTTPRoute", "httpbin", echoRoute, 1)

			time.Sleep(time.Second * 6)

			_ = s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK)

			echoLogs := s.GetDeploymentLogs("echo")
			Expect(echoLogs).To(ContainSubstring("GET /headers"))
		})
	})
	/*
		Context("HTTPRoute Status Updated", func() {
		})

		Context("HTTPRoute ParentRefs With Multiple Gateway", func() {
		})

		Context("HTTPRoute Canary", func() {
		})

		Context("HTTPRoute BackendRefs Discovery", func() {
		})
	*/
})
