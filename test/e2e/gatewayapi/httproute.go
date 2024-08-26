package gatewayapi

import (
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
  name: api7
spec:
  controllerName: "gateway.api7.io/api7-ingress-controller"
`

	var defautlGateway = `
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

	Context("HTTPRoute Base", func() {

		var defaultHTTPRouteWithHTTPBinGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: gateway1
  hostnames:
  - backends.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		It("Create/Updtea/Delete HTTPRoute", func() {
			By("create GatewayClass")
			err := s.CreateResourceFromStringWithNamespace(defautlGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("check GatewayClass condition")
			gcyaml, err := s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

			By("create Gateway")
			err = s.CreateResourceFromString(defautlGateway)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err := s.GetResourceYaml("Gateway", "api7ee")
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
			Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the api7-ingress-controller"), "checking Gateway condition message")

			By("create HTTPRoute")
			err = s.CreateResourceFromString(defaultHTTPRouteWithHTTPBinGet)
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")
			time.Sleep(5 * time.Second)

			By("check HTTPRoute condition")
			hryaml, err := s.GetResourceYaml("HTTPRoute", "httpbin")
			Expect(err).NotTo(HaveOccurred(), "getting HTTPRoute yaml")
			Expect(hryaml).To(ContainSubstring(`status: "True"`), "checking HTTPRoute condition status")

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				Expect().
				Status(200)
		})
	})

	/*
		Context("HTTPRoute Rule Match", func() {
		})


		Context("HTTPRoute Filter", func() {
		})

		Context("HTTPRoute Negative", func() {
		})

		Context("HTTPRoute Status Updated", func() {
		})

	*/
})
