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
		ControllerName: "apisix.apache.org/api7-ingress-controller",
	})

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: api7-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

	Context("Gateway", func() {
		var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "apisix.apache.org/api7-ingress-controller"
`

		var defaultGateway = `
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
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: api7-proxy-config
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
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: api7-proxy-config
`

		It("Create Gateway", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("check GatewayClass condition")
			gcyaml, err := s.GetResourceYaml("GatewayClass", "api7")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the api7-ingress-controller"), "checking GatewayClass condition message")

			By("create Gateway")
			err = s.CreateResourceFromString(defaultGateway)
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
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create secret")
			secretName := _secretName
			host := "api6.com"
			createSecret(s, secretName)
			var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "apisix.apache.org/api7-ingress-controller"
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
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: api7-proxy-config
`, host, secretName)
			By("create GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
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
			FIt("Check if SSL resource was created and updated", func() {
				By("create GatewayProxy")
				gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.DashboardTLSEndpoint, s.AdminKey())
				err := s.CreateResourceFromString(gatewayProxy)
				Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
				time.Sleep(5 * time.Second)

				secretName := _secretName
				createSecret(s, secretName)
				var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "apisix.apache.org/api7-ingress-controller"
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
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: api7-proxy-config
`, secretName, secretName)
				By("create GatewayClass")
				err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
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
				assert.Equal(GinkgoT(), tls[0].Labels["k8s/controller-name"], "apisix.apache.org/api7-ingress-controller")

				By("update secret")
				err = s.NewKubeTlsSecret(secretName, framework.TestCert, framework.TestKey)
				Expect(err).NotTo(HaveOccurred(), "update secret")
				Eventually(func() string {
					tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
					Expect(err).NotTo(HaveOccurred(), "list ssl from dashboard")
					if len(tls) < 1 {
						return ""
					}
					return tls[0].Cert
				}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(framework.TestCert))
			})
		})
	})

})
