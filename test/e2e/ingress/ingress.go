package ingress

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

const _secretName = "test-ingress-tls"

var Cert = framework.TestServerCert

var Key = framework.TestServerKey

func createSecret(s *scaffold.Scaffold, secretName string) {
	err := s.NewKubeTlsSecret(secretName, Cert, Key)
	assert.Nil(GinkgoT(), err, "create secret error")
}

var _ = Describe("Test Ingress", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
	})

	Context("Basic Ingress Functionality", func() {
		var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
`

		var defaultIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress
spec:
  ingressClassName: api7
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: example-service
            port:
              number: 80
`

		It("Create Ingress", func() {
			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")
			time.Sleep(5 * time.Second)

			By("check Ingress status")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress")
			Expect(err).NotTo(HaveOccurred(), "getting Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("example.com"), "checking Ingress host")
		})
	})

	Context("Ingress TLS", func() {
		It("Check if SSL resource was created", func() {
			secretName := _secretName
			host := "secure.example.com"
			createSecret(s, secretName)

			var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
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
            name: secure-service
            port:
              number: 80
`, host, secretName, host)

			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress with TLS")
			err = s.CreateResourceFromString(tlsIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with TLS")
			time.Sleep(5 * time.Second)

			By("check TLS configuration")
			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.NotEmpty(GinkgoT(), tls, "tls list should not be empty")

			// At least one TLS certificate should contain our host
			foundHost := false
			for _, sslObj := range tls {
				for _, sni := range sslObj.Snis {
					if sni == host {
						foundHost = true
						break
					}
				}
				if foundHost {
					break
				}
			}
			assert.True(GinkgoT(), foundHost, "host not found in any SSL configuration")
		})
	})

	Context("Multiple Paths and Backends", func() {
		var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
`

		var multiPathIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-multi
spec:
  ingressClassName: api7
  rules:
  - host: multi.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
      - path: /web
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80
      - path: /admin
        pathType: Prefix
        backend:
          service:
            name: admin-service
            port:
              number: 80
`

		It("Create Multi-path Ingress", func() {
			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Multi-path Ingress")
			err = s.CreateResourceFromString(multiPathIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Multi-path Ingress")
			time.Sleep(5 * time.Second)

			By("check Ingress status")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress-multi")
			Expect(err).NotTo(HaveOccurred(), "getting Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("multi.example.com"), "checking Ingress host")
			Expect(ingressYaml).To(ContainSubstring("/api"), "checking path /api")
			Expect(ingressYaml).To(ContainSubstring("/web"), "checking path /web")
			Expect(ingressYaml).To(ContainSubstring("/admin"), "checking path /admin")
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
`

		var secondaryIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7-secondary
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
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
            name: default-service
            port:
              number: 80
`

		var specificIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-specific
spec:
  ingressClassName: api7-secondary
  rules:
  - host: specific.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: specific-service
            port:
              number: 80
`

		It("Test IngressClass Selection", func() {
			By("create Default IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Secondary IngressClass")
			err = s.CreateResourceFromStringWithNamespace(secondaryIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Secondary IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress without IngressClass")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("check Default Ingress")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress-default")
			Expect(err).NotTo(HaveOccurred(), "getting Default Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("default.example.com"), "checking Default Ingress host")

			By("create Ingress with specific IngressClass")
			err = s.CreateResourceFromString(specificIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with specific IngressClass")
			time.Sleep(5 * time.Second)

			By("check Specific Ingress")
			ingressYaml, err = s.GetResourceYaml("Ingress", "api7-ingress-specific")
			Expect(err).NotTo(HaveOccurred(), "getting Specific Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("specific.example.com"), "checking Specific Ingress host")
			Expect(ingressYaml).To(ContainSubstring("api7-secondary"), "checking Specific Ingress class")
		})
	})
})
