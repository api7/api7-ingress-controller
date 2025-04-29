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
		const defaultGateway = `
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

		It("Delete GatewayClass", func() {
			By("create default GatewayClass")
			err := s.CreateResourceFromStringWithNamespace(defautlGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			Eventually(func() string {
				spec, err := s.GetResourceYaml("GatewayClass", "api7")
				Expect(err).NotTo(HaveOccurred(), "get resource yaml")
				return spec
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`status: "True"`))

			By("create a Gateway")
			err = s.CreateResourceFromStringWithNamespace(defaultGateway, s.CurrentNamespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(time.Second)

			By("try to delete the GatewayClass")
			_, err = s.RunKubectlAndGetOutput("delete", "GatewayClass", "api7", "--wait=false")
			Expect(err).NotTo(HaveOccurred())

			_, err = s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred(), "get resource yaml")

			output, err := s.RunKubectlAndGetOutput("describe", "GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred(), "describe GatewayClass api7")
			Expect(output).To(And(
				ContainSubstring("Warning"),
				ContainSubstring("DeletionBlocked"),
				ContainSubstring("gatewayclass-controller"),
				ContainSubstring("the GatewayClass is still used by Gateways"),
			))

			By("delete the Gateway")
			err = s.DeleteResource("Gateway", "api7ee")
			Expect(err).NotTo(HaveOccurred(), "deleting Gateway")
			time.Sleep(time.Second)

			By("try to delete the GatewayClass again")
			err = s.DeleteResource("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred())

			_, err = s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).To(HaveOccurred(), "get resource yaml")
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})
