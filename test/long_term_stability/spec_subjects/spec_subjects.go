package spec_subjects

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("API7 Ingress Controller Long Term Stability Tests", Ordered, func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName:       "gateway.api7.io/api7-ingress-controller",
			GinkgoBeforeCallback: BeforeAll,
			GinkgoAfterCallback:  AfterAll,
			KubectlLogger:        logger.Discard, // too many logs in long-term stability test so discard kubectl apply logs
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
    - path:
        type: Exact
        value: /get
    - path:
        type: Exact
        value: /post
    - path:
        type: Exact
        value: /image
    
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		httpRouteTemplate2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  labels:
    template_name: httpRouteTemplate2
spec:
  parentRefs:
  - name: api7ee
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: %s
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
			By("scale backend pods replicas")
			err := s.ScaleHTTPBIN(5)
			Expect(err).NotTo(HaveOccurred(), "scaling httpbin")
		})

		It("rolling update", func() {
			var total = 5
			for i := 0; i < total; i++ {
				By(fmt.Sprintf("rolling update deployment/httpbin-deployment-e2e-test [%02d/%02d]", i+1, total))
				now := time.Now().Format("2006_01_02_15_04_05")
				_, err := s.RunKubectlAndGetOutput("set", "env", "deployment/httpbin-deployment-e2e-test", "ENV_NOW="+now)
				Expect(err).NotTo(HaveOccurred(), "kubectl set env deployment/httpbin-deployment-e2e-test MOCK_ENV=%s", now)
				time.Sleep(time.Minute)
			}

			err := s.DownloadLocustReport(fmt.Sprintf("%02d_rolling_update", it))
			Expect(err).NotTo(HaveOccurred(), "getting locust report")
		})

		It("scale replicas", func() {
			var total = 5
			for i := 0; i < total; i++ {
				By(fmt.Sprintf("scale replicas for deployment/httpbin-deployment-e2e-test [%02d/%02d]", i+1, total))
				err := s.ScaleHTTPBIN(4 + i%2*3) // scale to 4 if "i" is even, scale to 7 if "i" is odd.
				Expect(err).NotTo(HaveOccurred(), "scale replicas for deployment/httpbin-deployment-e2e-test")
				time.Sleep(time.Minute)
			}

			err := s.DownloadLocustReport(fmt.Sprintf("%02d_scale_replicas", it))
			Expect(err).NotTo(HaveOccurred(), "getting locust report")
		})
	})

	Context("Large-scale HTTPRoute", func() {
		var (
			reconcileDurationPerHTTPRoute = 3 * time.Second
			probeDuration                 = 100 * time.Millisecond

			resourceTypeHTTPRoute = "HTTPRoute"
			label                 = "template_name=httpRouteTemplate2"
		)

		for _, total := range []int{500, 2000, 5000} {
			var (
				reconcileDurationBatchProcess = time.Duration(total) * reconcileDurationPerHTTPRoute
				title                         = strconv.FormatInt(int64(total), 10) + " HTTPRoute"
			)

			It(title, func() {
				defer func() {
					By("cleaning up HTTPRoutes")
					err := s.DeleteResourcesByLabels(resourceTypeHTTPRoute, label)
					Expect(err).NotTo(HaveOccurred(), "delete HTTPRoute by label")

					Eventually(func() string {
						output, err := s.GetResourcesByLabelsOutput(resourceTypeHTTPRoute, label)
						Expect(err).NotTo(HaveOccurred(), "getting HTTPRoute")
						return output
					}).WithTimeout(reconcileDurationBatchProcess).ProbeEvery(probeDuration).
						Should(ContainSubstring("No resources found"))
				}()

				By("prepare HTTPRoutes")
				for i := 0; i < total+100; i++ {
					By(fmt.Sprintf("prepare HTTPRoutes [%04d/%04d]", i+1, total))
					routeName := "httpbin-" + strconv.FormatInt(int64(i), 10)
					pathValue := "/delay/" + strconv.FormatInt(int64(i), 10)
					err := s.CreateResourceFromString(fmt.Sprintf(httpRouteTemplate2, routeName, pathValue))
					Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")

					message := retry.DoWithRetry(s.GinkgoT, "Wait for HTTPRoute ok", 100, time.Second, func() (string, error) {
						yaml_, err := s.GetResourceYaml(resourceTypeHTTPRoute, routeName)
						if err != nil {
							return "", err
						}
						if !strings.Contains(yaml_, `status: "True"`) {
							return "", errors.New("HTTPRoute status is not True")
						}
						return "HTTPRoute is now available", nil
					},
					)
					s.Logf(message)
				}

				By("delete 100 HTTPRoutes")
				for i := total; i < total+100; i++ {
					By(fmt.Sprintf("prepare 1000 HTTPRoute [%04d/%04d]", i, total+100))
					routeName := "httpbin-" + strconv.FormatInt(int64(i), 10)
					err := s.DeleteResource(resourceTypeHTTPRoute, routeName)
					Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")

					Eventually(func() string {
						_, err := s.GetResourceYaml(resourceTypeHTTPRoute, "")
						return err.Error()
					}).WithTimeout(reconcileDurationPerHTTPRoute).ProbeEvery(probeDuration).
						Should(ContainSubstring("not found"))
				}

				err := s.DownloadLocustReport(fmt.Sprintf("%02d_large_scale_httproute(1000)", it))
				Expect(err).NotTo(HaveOccurred(), "getting locust report")
			})
		}
	})

	PContext("Ingress Controller is crashing", func() {
		It("it 0", func() {
			Ω(true).Should(BeTrue())
		})
	})

	PContext("Regression Tests", func() {
		// Under large-scale HTTPRoute, some HTTPRoute resources are abnormal, which cannot affect the synchronization
		// efficient of other resources
		It("", func() {})

		// Under large-scale CRDs, some CRDs resources are abnormal, which cannot affect the synchronization efficient
		// of other resources
		It("", func() {})

		// When a large number of CRDs are applied concurrently, the processing capacity of IngressController is linear
		// (≤O(n), n is the number of resources applied simultaneously)
		It("", func() {})

		// Under large-scale CRDs, some CRDs are added/deleted/modified concurrently, and the processing capacity of
		// IngressController has no obvious relationship with the number of existing CRDs
		It("", func() {})

		// When IngressController is unexpectedly unavailable, it does not affect the existing configuration of the
		// data plane
		It("", func() {})
	})
})
