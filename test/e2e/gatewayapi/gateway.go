package gatewayapi

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

const _secretName = "test-apisix-tls"

var Cert = strings.TrimSpace(framework.TestServerCert)

var Key = strings.TrimSpace(framework.TestServerKey)

func createSecret(s *scaffold.Scaffold, secretName string) {
	err := s.NewKubeTlsSecret(secretName, Cert, Key)
	assert.Nil(GinkgoT(), err, "create secret error")
}

var _ = Describe("Test Gateway", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
	})

	Context("Gateway", func() {
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

		var noClassGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: api7ee-not-class
spec:
  gatewayClassName: api7-not-exist
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
`

		It("Create Gateway", func() {
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

			By("create Gateway with not accepted GatewayClass")
			err = s.CreateResourceFromString(noClassGateway)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err = s.GetResourceYaml("Gateway", "api7ee-not-class")
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: Unknown`), "checking Gateway condition status")
		})
	})

	Context("Gateway SSL", func() {
		It("Check if SSL resource was created", func() {
			secretName := _secretName
			host := "api6.com"
			createSecret(s, secretName)
			var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "gateway.api7.io/api7-ingress-controller"
`

			var defaultGateway = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: api7ee
spec:
  gatewayClassName: api7
  listeners:
    - name: http1
      protocol: HTTPS
      port: 443
      hostname: %s
      tls:
        certificateRefs:
        - kind: Secret
          group: ""
          name: %s
`, host, secretName)
			By("create GatewayClass")
			err := s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromString(defaultGateway)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(10 * time.Second)

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Cert, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host, "*.api6.com"}, tls[0].Snis)
		})

		Context("Gateway SSL with and without hostname", func() {
			It("Check if SSL resource was created", func() {
				secretName := _secretName
				createSecret(s, secretName)
				var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "gateway.api7.io/api7-ingress-controller"
`

				var defaultGateway = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: same-namespace-with-https-listener
spec:
  gatewayClassName: api7
  listeners:
  - name: https
    port: 443
    protocol: HTTPS
    allowedRoutes:
      namespaces:
        from: Same
    tls:
      certificateRefs:
      - group: ""
        kind: Secret
        name: %s
  - name: https-with-hostname
    port: 443
    hostname: api6.com
    protocol: HTTPS
    allowedRoutes:
      namespaces:
        from: Same
    tls:
      certificateRefs:
      - group: ""
        kind: Secret
        name: %s
`, secretName, secretName)
				By("create GatewayClass")
				err := s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
				Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
				time.Sleep(5 * time.Second)

				By("create Gateway")
				err = s.CreateResourceFromString(defaultGateway)
				Expect(err).NotTo(HaveOccurred(), "creating Gateway")
				time.Sleep(10 * time.Second)

				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				assert.Nil(GinkgoT(), err, "list tls error")
				assert.Len(GinkgoT(), tls, 1, "tls number not expect")
				assert.Equal(GinkgoT(), Cert, tls[0].Cert, "tls cert not expect")
				assert.Equal(GinkgoT(), tls[0].Labels["k8s/controller-name"], "gateway.api7.io/api7-ingress-controller")
			})
		})
	})

})
