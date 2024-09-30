package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Check if controller cache gets synced with correct resources", func() {
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
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
`

	var ResourceApplied = func(s *scaffold.Scaffold, resourType, resourceName, resourceRaw string, observedGeneration int) {
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
	var beforeEach = func(s *scaffold.Scaffold, gatewayName string) {
		By(fmt.Sprintf("create GatewayClass for controller %s", s.GetControllerName()))
		gatewayClassName := fmt.Sprintf("api7-%d", time.Now().Unix())
		err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defautlGatewayClass, gatewayClassName, s.GetControllerName()), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(20 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defautlGateway, gatewayName, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(20 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", gatewayName)
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the api7-ingress-controller"), "checking Gateway condition message")
	}

	Context("Create resource with first controller", func() {
		s1 := scaffold.NewScaffold(&scaffold.Options{
			Name:           "gateway1",
			ControllerName: "gateway.api7.io/api7-ingress-controller-1",
		})
		s2 := scaffold.NewScaffold(&scaffold.Options{
			Name:           "gateway2",
			ControllerName: "gateway.api7.io/api7-ingress-controller-2",
		})
		var route1 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: gateway1
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
 `
		var route2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin2
spec:
  parentRefs:
  - name: gateway2
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
`
		BeforeEach(func() {
			beforeEach(s1, "gateway1")
			beforeEach(s2, "gateway2")
		})
		It("Apply resource ", func() {
			ResourceApplied(s1, "HTTPRoute", "httpbin", route1, 1)
			ResourceApplied(s2, "HTTPRoute", "httpbin2", route2, 1)
			routes, err := s1.DefaultDataplaneResource().Route().List(s1.Context)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			assert.Equal(GinkgoT(), routes[0].Labels["controller_name"], "gateway.api7.io/api7-ingress-controller-1")

			routes, err = s2.DefaultDataplaneResource().Route().List(s2.Context)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			assert.Equal(GinkgoT(), routes[0].Labels["controller_name"], "gateway.api7.io/api7-ingress-controller-2")
		})
	})
})
