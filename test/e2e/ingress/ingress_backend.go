package ingress

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Ingress Backend Services", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
	})

	Context("Handling Non-existent Backend Services", func() {
		var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
`

		var nonExistentServiceIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-non-existent
spec:
  ingressClassName: api7
  rules:
  - host: non-existent.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: non-existent-service
            port:
              number: 80
`

		It("Creates Ingress with Non-existent Backend Service", func() {
			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress with Non-existent Backend Service")
			err = s.CreateResourceFromString(nonExistentServiceIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with Non-existent Backend Service")
			time.Sleep(5 * time.Second)

			By("check Ingress status")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress-non-existent")
			Expect(err).NotTo(HaveOccurred(), "getting Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("non-existent.example.com"), "checking Ingress host")
		})
	})

	Context("Update Backend Service", func() {
		var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: api7
spec:
  controller: "gateway.api7.io/api7-ingress-controller"
`

		var serviceYaml = `
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: backend
`

		var updatedServiceYaml = `
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  ports:
  - port: 80
    targetPort: 9090
    protocol: TCP
    name: http
  - port: 443
    targetPort: 8443
    protocol: TCP
    name: https
  selector:
    app: backend-updated
`

		var ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api7-ingress-backend
spec:
  ingressClassName: api7
  rules:
  - host: backend.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: backend-service
            port:
              number: 80
`

		It("Updates Backend Service for Ingress", func() {
			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Backend Service")
			err = s.CreateResourceFromString(serviceYaml)
			Expect(err).NotTo(HaveOccurred(), "creating Backend Service")
			time.Sleep(5 * time.Second)

			By("create Ingress with Backend Service")
			err = s.CreateResourceFromString(ingressYaml)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with Backend Service")
			time.Sleep(5 * time.Second)

			By("check Ingress status")
			ingressYaml, err := s.GetResourceYaml("Ingress", "api7-ingress-backend")
			Expect(err).NotTo(HaveOccurred(), "getting Ingress yaml")
			Expect(ingressYaml).To(ContainSubstring("backend.example.com"), "checking Ingress host")

			By("update Backend Service")
			err = s.CreateResourceFromString(updatedServiceYaml)
			Expect(err).NotTo(HaveOccurred(), "updating Backend Service")
			time.Sleep(10 * time.Second)

			By("check Service after update")
			serviceYaml, err := s.GetResourceYaml("Service", "backend-service")
			Expect(err).NotTo(HaveOccurred(), "getting Service yaml")
			Expect(serviceYaml).To(ContainSubstring("9090"), "checking updated Service targetPort")
			Expect(serviceYaml).To(ContainSubstring("443"), "checking added Service port")
		})
	})
})
