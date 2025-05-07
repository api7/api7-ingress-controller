// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package conformance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
)

var gatewayClassName = "api7"
var controllerName = "apisix.apache.org/api7-ingress-controller"

var gatewayClass = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`, gatewayClassName, controllerName)

var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: conformance-gateway-proxy
  namespace: %s
spec:
  statusAddress:
    - %s
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
        - %s
      auth:
        type: AdminKey
        adminKey:
          value: %s
`

type GatewayProxyOpts struct {
	StatusAddress string
	AdminKey      string
	AdminEndpoint string
}

var defaultGatewayProxyOpts GatewayProxyOpts

func deleteNamespace(kubectl *k8s.KubectlOptions) {
	// gateway api conformance test namespaces
	namespacesToDelete := []string{
		"gateway-conformance-infra",
		"gateway-conformance-web-backend",
		"gateway-conformance-app-backend",
		"api7ee-conformance-test",
	}

	for _, ns := range namespacesToDelete {
		_, err := k8s.GetNamespaceE(GinkgoT(), kubectl, ns)
		if err == nil {
			// Namespace exists, delete it
			GinkgoT().Logf("Deleting existing namespace: %s", ns)
			err := k8s.DeleteNamespaceE(GinkgoT(), kubectl, ns)
			if err != nil {
				GinkgoT().Logf("Error deleting namespace %s: %v", ns, err)
				continue
			}

			// Wait for deletion to complete by checking until GetNamespaceE returns an error
			_, err = retry.DoWithRetryE(
				GinkgoT(),
				fmt.Sprintf("Waiting for namespace %s to be deleted", ns),
				30,
				5*time.Second,
				func() (string, error) {
					_, err := k8s.GetNamespaceE(GinkgoT(), kubectl, ns)
					if err != nil {
						// Namespace is gone, which is what we want
						return "Namespace deleted", nil
					}
					return "", fmt.Errorf("namespace %s still exists", ns)
				},
			)

			if err != nil {
				GinkgoT().Logf("Error waiting for namespace %s to be deleted: %v", ns, err)
			}
		} else {
			GinkgoT().Logf("Namespace %s does not exist or cannot be accessed", ns)
		}
	}
}

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	f := framework.NewFramework()

	f.BeforeSuite()

	// Check and delete specific namespaces if they exist
	kubectl := k8s.NewKubectlOptions("", "", "default")
	deleteNamespace(kubectl)

	namespace := "api7ee-conformance-test"

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
		ControllerName: "apisix.apache.org/api7-ingress-controller",
		AdminKey:       adminKey,
		AdminTLSVerify: false,
		Namespace:      namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint,
		StatusAddress:  address,
	})

	defaultGatewayProxyOpts = GatewayProxyOpts{
		StatusAddress: address,
		AdminKey:      adminKey,
		AdminEndpoint: framework.DashboardTLSEndpoint,
	}

	patchGatewaysForConformanceTest(context.Background(), f.K8sClient)

	code := m.Run()

	f.AfterSuite()

	os.Exit(code)
}

func patchGatewaysForConformanceTest(ctx context.Context, k8sClient client.Client) {
	var gatewayProxyMap = make(map[string]bool)

	// list all gateways and patch them
	patchGateway := func(ctx context.Context, k8sClient client.Client) bool {
		gatewayList := &gatewayv1.GatewayList{}
		if err := k8sClient.List(ctx, gatewayList); err != nil {
			return false
		}

		patched := false
		for i := range gatewayList.Items {
			gateway := &gatewayList.Items[i]

			// check if the gateway already has infrastructure.parametersRef
			if gateway.Spec.Infrastructure != nil &&
				gateway.Spec.Infrastructure.ParametersRef != nil {
				continue
			}

			GinkgoT().Logf("Patching Gateway %s", gateway.Name)
			// check if the gateway proxy has been created, if not, create it
			if !gatewayProxyMap[gateway.Namespace] {
				gatewayProxy := fmt.Sprintf(gatewayProxyYaml,
					gateway.Namespace,
					defaultGatewayProxyOpts.StatusAddress,
					defaultGatewayProxyOpts.AdminEndpoint,
					defaultGatewayProxyOpts.AdminKey)
				kubectl := k8s.NewKubectlOptions("", "", gateway.Namespace)
				k8s.KubectlApplyFromString(GinkgoT(), kubectl, gatewayProxy)

				// Mark this namespace as having a GatewayProxy
				gatewayProxyMap[gateway.Namespace] = true
			}

			// add infrastructure.parametersRef
			gateway.Spec.Infrastructure = &gatewayv1.GatewayInfrastructure{
				ParametersRef: &gatewayv1.LocalParametersReference{
					Group: "apisix.apache.org",
					Kind:  "GatewayProxy",
					Name:  "conformance-gateway-proxy",
				},
			}

			if err := k8sClient.Update(ctx, gateway); err != nil {
				GinkgoT().Logf("Failed to patch Gateway %s: %v", gateway.Name, err)
				continue
			}

			patched = true
			GinkgoT().Logf("Successfully patched Gateway %s with GatewayProxy reference", gateway.Name)
		}

		return patched
	}

	// continuously monitor and patch gateway resources
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// clean up the gateway proxy
				for namespace := range gatewayProxyMap {
					kubectl := k8s.NewKubectlOptions("", "", namespace)
					_ = k8s.RunKubectlE(GinkgoT(), kubectl, "delete", "gatewayproxy", "conformance-gateway-proxy")
				}
				return
			case <-ticker.C:
				patchGateway(ctx, k8sClient)
			}
		}
	}()
}
