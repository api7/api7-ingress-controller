package ingress

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

const _secretName = "test-ingress-tls"

var Cert = strings.TrimSpace(framework.TestServerCert)

var Key = strings.TrimSpace(framework.TestServerKey)

func createSecret(s *scaffold.Scaffold, secretName string) {
	err := s.NewKubeTlsSecret(secretName, Cert, Key)
	assert.Nil(GinkgoT(), err, "create secret error")
}

var _ = Describe("Test Ingress", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
	})

	var gatewayProxyYaml = `
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

	Context("Ingress TLS", func() {
		It("Check if SSL resource was created", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())

			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			secretName := _secretName
			host := "api6.com"
			createSecret(s, secretName)

			var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
  parameters:
    apiGroup: "gateway.apisix.io"
    kind: "GatewayProxy"
    name: "api7-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

			var tlsIngress = fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-tls
spec:
  ingressClassName: api7
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`, host, secretName, host)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress with TLS")
			err = s.CreateResourceFromString(tlsIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with TLS")
			time.Sleep(5 * time.Second)

			By("check TLS configuration")
			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Cert, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host}, tls[0].Snis)
		})
	})

	Context("IngressClass Selection", func() {
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
  - host: default.example.com
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

		It("Test IngressClass Selection", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())

			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Default IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress without IngressClass")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("verify default ingress")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("default.example.com").
				Expect().
				Status(200)
		})
	})

	Context("IngressClass with GatewayProxy", func() {
		gatewayProxyYaml := `
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

		gatewayProxyWithSecretYaml := `
apiVersion: gateway.apisix.io/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config-with-secret
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
          valueFrom:
            secretKeyRef:
              name: admin-secret
              key: admin-key
`

		var ingressClassWithProxy = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7-with-proxy
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

		var ingressClassWithProxySecret = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7-with-proxy-secret
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
  parameters:
    apiGroup: "gateway.apisix.io"
    kind: "GatewayProxy"
    name: "api7-proxy-config-with-secret"
    namespace: "default"
    scope: "Namespace"
`

		var testIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-with-proxy
spec:
  ingressClassName: api7-with-proxy
  rules:
  - host: proxy.example.com
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

		var testIngressWithSecret = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-with-proxy-secret
spec:
  ingressClassName: api7-with-proxy-secret
  rules:
  - host: proxy-secret.example.com
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

		It("Test IngressClass with GatewayProxy", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())

			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass with GatewayProxy reference")
			err = s.CreateResourceFromStringWithNamespace(ingressClassWithProxy, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Ingress with GatewayProxy IngressClass")
			err = s.CreateResourceFromString(testIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
			time.Sleep(5 * time.Second)

			By("verify HTTP request")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("proxy.example.com").
				Expect().
				Status(200)
		})

		It("Test IngressClass with GatewayProxy using Secret", func() {
			By("create admin key secret")
			adminSecret := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: admin-secret
  namespace: default
type: Opaque
stringData:
  admin-key: %s
`, s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(adminSecret, "default")
			Expect(err).NotTo(HaveOccurred(), "creating admin secret")
			time.Sleep(5 * time.Second)

			By("create GatewayProxy with Secret reference")
			gatewayProxy := fmt.Sprintf(gatewayProxyWithSecretYaml, framework.DashboardTLSEndpoint)
			err = s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with Secret")
			time.Sleep(5 * time.Second)

			By("create IngressClass with GatewayProxy reference")
			err = s.CreateResourceFromStringWithNamespace(ingressClassWithProxySecret, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Ingress with GatewayProxy IngressClass")
			err = s.CreateResourceFromString(testIngressWithSecret)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
			time.Sleep(5 * time.Second)

			By("verify HTTP request")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("proxy-secret.example.com").
				Expect().
				Status(200)
		})
	})
})
