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
            name: httpbin-service-e2e-test
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

			By("verify HTTP request")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("example.com").
				Expect().
				Status(200)
		})
	})

	Context("Ingress TLS", func() {
		It("Check if SSL resource was created", func() {
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
			By("create Default IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress without IngressClass")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("check Default Ingress")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress-default")
			Expect(err).NotTo(HaveOccurred(), "getting Default Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("default.example.com"), "checking Default Ingress host")

			By("verify default ingress")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("default.example.com").
				Expect().
				Status(200)
		})
	})
})
