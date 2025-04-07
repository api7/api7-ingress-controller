package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Consumer", func() {
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
apiVersion: gateway.apisix.io/v1alpha1
kind: PluginConfig
metadata:
  name: auth-plugin-config
spec:
  plugins:
    - name: multi-auth
      config:
        auth_plugins:
          - basic-auth: {}
          - key-auth:
              header: apikey
---

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
    filters:
    - type: ExtensionRef
      extensionRef:
        group: gateway.api7.io
        kind: PluginConfig
        name: auth-plugin-config
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	var beforeEachHTTP = func() {
		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("api7-%d", time.Now().Unix())
		gatewayString := fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName())
		err := s.CreateResourceFromStringWithNamespace(gatewayString, "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(5 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(
			ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"),
			"checking GatewayClass condition message",
		)

		By("create Gateway")
		err = s.CreateResourceFromString(fmt.Sprintf(defaultGateway, gatewayClassName))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", "api7ee")
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(
			ContainSubstring("message: the gateway has been accepted by the api7-ingress-controller"),
			"checking Gateway condition message",
		)

		s.ResourceApplied("httproute", "httpbin", defaultHTTPRoute, 1)
	}

	Context("Consumer plugins", func() {
		var limitCountConsumer = `
apiVersion: gateway.apisix.io/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: api7ee
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key
  plugins:
    - name: limit-count
      config:
        count: 2
        time_window: 60
        rejected_code: 503
        key: remote_addr
`

		var unlimitConsumer = `
apiVersion: gateway.apisix.io/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample2
spec:
  gatewayRef:
    name: api7ee
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key2
`

		BeforeEach(beforeEachHTTP)

		It("limit-count plugin", func() {
			s.ResourceApplied("Consumer", "consumer-sample", limitCountConsumer, 1)
			s.ResourceApplied("Consumer", "consumer-sample2", unlimitConsumer, 1)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			By("trigger limit-count")
			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(503)

			for i := 0; i < 10; i++ {
				s.NewAPISIXClient().
					GET("/get").
					WithHeader("apikey", "sample-key2").
					WithHost("httpbin.org").
					Expect().
					Status(200)
			}
		})
	})

	Context("Credential", func() {
		var defaultCredential = `apiVersion: gateway.apisix.io/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: api7ee
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      config:
        username: sample-user
        password: sample-password
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key
    - type: key-auth
      name: key-auth-sample2
      config:
        key: sample-key2
`
		var updateCredential = `apiVersion: gateway.apisix.io/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: api7ee
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      config:
        username: sample-user
        password: sample-password
  plugins:
    - name: key-auth
      config:
        key: consumer-key
`
		BeforeEach(beforeEachHTTP)

		It("Create/Update/Delete", func() {
			s.ResourceApplied("Consumer", "consumer-sample", defaultCredential, 1)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key2").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			By("update Consumer")
			s.ResourceApplied("Consumer", "consumer-sample", updateCredential, 2)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(401)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key2").
				WithHost("httpbin.org").
				Expect().
				Status(401)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "consumer-key").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			By("delete Consumer")
			err := s.DeleteResourceFromString(updateCredential)
			Expect(err).NotTo(HaveOccurred(), "deleting Consumer")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(401)
		})
	})

	Context("SecretRef", func() {
		var keyAuthSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: key-auth-secret
data:
  key: c2FtcGxlLWtleQ==
`
		var basicAuthSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: basic-auth-secret
data:
  username: c2FtcGxlLXVzZXI=
  password: c2FtcGxlLXBhc3N3b3Jk
`
		var defaultConsumer = `
apiVersion: gateway.apisix.io/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: api7ee
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      secretRef:
        name: basic-auth-secret
    - type: key-auth
      name: key-auth-sample
      secretRef:
        name: key-auth-secret
    - type: key-auth
      name: key-auth-sample2
      config:
        key: sample-key2
`
		BeforeEach(beforeEachHTTP)

		It("Create/Update/Delete", func() {
			err := s.CreateResourceFromString(keyAuthSecret)
			Expect(err).NotTo(HaveOccurred(), "creating key-auth secret")
			err = s.CreateResourceFromString(basicAuthSecret)
			Expect(err).NotTo(HaveOccurred(), "creating basic-auth secret")
			s.ResourceApplied("Consumer", "consumer-sample", defaultConsumer, 1)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(200)

			By("delete consumer")
			err = s.DeleteResourceFromString(defaultConsumer)
			Expect(err).NotTo(HaveOccurred(), "deleting consumer")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/get").
				WithHeader("apikey", "sample-key").
				WithHost("httpbin.org").
				Expect().
				Status(401)

			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(401)
		})
	})
})
