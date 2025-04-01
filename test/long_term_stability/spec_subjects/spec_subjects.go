package spec_subjects

import (
	"fmt"
	"net/http"
	"time"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("API7 Ingress Controller Long Term Stability Tests", Ordered, func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName:       "gateway.api7.io/api7-ingress-controller",
			GinkgoBeforeCallback: BeforeAll,
			GinkgoAfterCallback:  AfterAll,
		})
		it int
	)
	var (
		gatewayClassTemplate = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: gateway.api7.io/api7-ingress-controller
`
		gatewayTemplate = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: api7ee
spec:
  gatewayClassName: api7
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
`
		httpRouteTemplate = `
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
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
	)

	BeforeAll(func() {
		By("apply GatewayClass")
		err := s.CreateResourceFromString(gatewayClassTemplate)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		Eventually(func() string {
			yaml_, err := s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred())
			return yaml_
		}).WithTimeout(8*time.Second).ProbeEvery(time.Second).
			Should(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")

		By("apply Gateway")
		err = s.CreateResourceFromString(gatewayTemplate)
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		Eventually(func() string {
			yaml_, err := s.GetResourceYaml("Gateway", "api7ee")
			Expect(err).NotTo(HaveOccurred())
			return yaml_
		}).WithTimeout(8*time.Second).ProbeEvery(time.Second).
			Should(ContainSubstring(`status: "True"`), "checking Gateway condition status")

		By("deploy locust")
		_ = s.DeployLocust()
	})

	BeforeEach(func() {
		By("Create HTTPRoute")
		err := s.CreateResourceFromString(httpRouteTemplate)
		Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")
		Eventually(func() string {
			yaml_, err := s.GetResourceYaml("HTTPRoute", "httpbin")
			Expect(err).NotTo(HaveOccurred(), "getting yaml: %s", yaml_)
			return yaml_
		}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
			Should(ContainSubstring(`status: "True"`))
		Eventually(func() int {
			return s.NewAPISIXClient().GET("/headers").WithHost("httpbin.example").Expect().Raw().StatusCode
		}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusOK))
	})

	BeforeEach(func() {
		it++
	})

	JustBeforeEach(func() {
		By("reset locust statistics")
		err := s.ResetLocust()
		Expect(err).NotTo(HaveOccurred(), "reset locust")
	})

	Context("Benchmark", func() {
		It("benchmark", func() {
			By("sleep and waiting for locust test")
			time.Sleep(10 * time.Minute)

			err := s.DownloadLocustReport(fmt.Sprintf("%02d_benchmark", it))
			Expect(err).NotTo(HaveOccurred(), "getting locust report")
		})
	})

	Context("Service Discovery", func() {
		BeforeEach(func() {
			// scale backend pods replicas
			err := s.ScaleHTTPBIN(5)
			Expect(err).NotTo(HaveOccurred(), "scaling httpbin")
		})

		It("service discovery", func() {
			var total = 5
			for i := 0; i < total; i++ {
				By(fmt.Sprintf("rolling update deployment/httpbin-deployment-e2e-test [%02d/%02d]", i+1, total))
				now := time.Now().Format("2006_01_02_15_04_05")
				_, err := s.RunKubectlAndGetOutput("set", "env", "deployment/httpbin-deployment-e2e-test", "ENV_NOW="+now)
				Expect(err).NotTo(HaveOccurred(), "kubectl set env deployment/httpbin-deployment-e2e-test MOCK_ENV=%s", now)
				time.Sleep(time.Minute)
			}

			err := s.DownloadLocustReport(fmt.Sprintf("%02d_service_discovery", it))
			Expect(err).NotTo(HaveOccurred(), "getting locust report")
		})
	})

	Context("Large-scale HTTPRoute", func() {
		It("it 0", func() {
			Ω(true).Should(BeTrue())
		})
	})

	Context("Ingress Controller is crashing", func() {
		It("it 0", func() {
			Ω(true).Should(BeTrue())
		})
	})

	Context("Regression Tests", func() {
		It("Under large-scale HTTPRoute, some HTTPRoute resources are abnormal, which cannot affect the synchronization efficient of other resources", func() {})

		It("Under large-scale CRDs, some CRDs resources are abnormal, which cannot affect the synchronization efficient of other resources", func() {})

		It("When a large number of CRDs are applied concurrently, the processing capacity of IngressController is linear (≤O(n), n is the number of resources applied simultaneously)", func() {})

		It("Under large-scale CRDs, some CRDs are added/deleted/modified concurrently, and the processing capacity of IngressController has no obvious relationship with the number of existing CRDs", func() {})

		It("When IngressController is unexpectedly unavailable, it does not affect the existing configuration of the data plane", func() {})
	})
})
