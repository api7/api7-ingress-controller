package gatewayapi

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayClass", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/api7-ingress-controller",
	})

	Context("Create GatewayClass", func() {
		var defautlGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "apisix.apache.org/api7-ingress-controller"
`

		var noGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7-not-accepeted
spec:
  controllerName: "apisix.apache.org/not-exist"
`
		It("Create GatewayClass", func() {
			By("create default GatewayClass")
			err := s.CreateResourceFromStringWithNamespace(defautlGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			gcyaml, err := s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

			By("create GatewayClass with not accepted")
			err = s.CreateResourceFromStringWithNamespace(noGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			gcyaml, err = s.GetResourceYaml("GatewayClass", "api7-not-accepeted")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: Unknown`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: Waiting for controller"), "checking GatewayClass condition message")
		})
	})
})
