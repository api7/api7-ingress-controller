package conformance

import (
	"fmt"
	"os"
	"testing"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/gruntwork-io/terratest/modules/k8s"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var gatewayClassName = "api7"
var controllerName = "gateway.api7.io/api7-ingress-controller"

var gatewayClass = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`, gatewayClassName, controllerName)

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	f := framework.NewFramework()

	f.BeforeSuite()

	namespace := "api7ee-conformance-test"

	kubectl := k8s.NewKubectlOptions("", "", "default")

	k8s.KubectlApplyFromString(GinkgoT(), kubectl, gatewayClass)
	defer k8s.KubectlDeleteFromString(GinkgoT(), kubectl, gatewayClass)
	k8s.CreateNamespace(GinkgoT(), kubectl, namespace)
	defer k8s.DeleteNamespace(GinkgoT(), kubectl, namespace)

	gatewayGouopId := f.CreateNewGatewayGroupWithIngress()
	adminKey := f.GetAdminKey(gatewayGouopId)

	svc := f.DeployGateway(framework.DataPlaneDeployOptions{
		Namespace:              namespace,
		GatewayGroupID:         gatewayGouopId,
		DPManagerEndpoint:      framework.DPManagerTLSEndpoint,
		SetEnv:                 true,
		SSLKey:                 framework.TestKey,
		SSLCert:                framework.TestCert,
		TLSEnabled:             true,
		ForIngressGatewayGroup: true,
		ServiceType:            "LoadBalancer",
		ServiceHTTPPort:        80,
		ServiceHTTPSPort:       443,
	})

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		Fail("No LoadBalancer found for the service")
	}

	address := svc.Status.LoadBalancer.Ingress[0].IP

	f.DeployIngress(framework.IngressDeployOpts{
		ControllerName: "gateway.api7.io/api7-ingress-controller",
		AdminKey:       adminKey,
		AdminTLSVerify: false,
		Namespace:      namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint + "/apisix/admin",
		StatusAddress:  address,
	})

	code := m.Run()

	f.AfterSuite()

	os.Exit(code)
}
