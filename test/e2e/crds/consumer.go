package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Consumer", func() {
	s := scaffold.NewDefaultScaffold()

	var defaultGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
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
      group: apisix.apache.org
      kind: GatewayProxy
      name: api7-proxy-config
`

	var defaultHTTPRoute = `
apiVersion: apisix.apache.org/v1alpha1
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
        group: apisix.apache.org
        kind: PluginConfig
        name: auth-plugin-config
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	Context("Consumer plugins", func() {
		var limitCountConsumer = `
apiVersion: apisix.apache.org/v1alpha1
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
apiVersion: apisix.apache.org/v1alpha1
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

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayProxy, defaultGatewayClass, defaultGateway, defaultHTTPRoute)
		})

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
		var defaultCredential = `
apiVersion: apisix.apache.org/v1alpha1
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
		var updateCredential = `apiVersion: apisix.apache.org/v1alpha1
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

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayProxy, defaultGatewayClass, defaultGateway, defaultHTTPRoute)
		})

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
apiVersion: apisix.apache.org/v1alpha1
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

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayProxy, defaultGatewayClass, defaultGateway, defaultHTTPRoute)
		})
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

	Context("Consumer with GatewayProxy Update", func() {
		var additionalGatewayGroupID string

		var defaultCredential = `
apiVersion: apisix.apache.org/v1alpha1
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
`
		var updatedGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
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

		BeforeEach(func() {
			s.ApplyDefaultGatewayResource(defaultGatewayProxy, defaultGatewayClass, defaultGateway, defaultHTTPRoute)
		})

		It("Should sync consumer when GatewayProxy is updated", func() {
			s.ResourceApplied("Consumer", "consumer-sample", defaultCredential, 1)

			// verify basic-auth works
			s.NewAPISIXClient().
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
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

			By("Consumer not found for additional gateway group")
			client.
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(404)

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, framework.DashboardTLSEndpoint, resources.AdminAPIKey)
			err = s.CreateResourceFromString(updatedProxy)
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")
			time.Sleep(30 * time.Second)

			By("verify Consumer works for additional gateway group")
			client.
				GET("/get").
				WithBasicAuth("sample-user", "sample-password").
				WithHost("httpbin.org").
				Expect().
				Status(200)
		})
	})
})
