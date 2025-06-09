// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scaffold

import (
	"fmt"
	"os"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/utils"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

type API7Deployer struct {
	*Scaffold

	gatewayGroupID string
}

func NewAPI7Deployer(s *Scaffold) *API7Deployer {
	return &API7Deployer{
		Scaffold: s,
	}
}

func (s *API7Deployer) BeforeEach() {
	var err error
	s.UploadLicense()
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
	}
	if s.opts.ControllerName == "" {
		s.opts.ControllerName = fmt.Sprintf("%s/%d", DefaultControllerName, time.Now().Nanosecond())
	}
	s.finalizers = nil
	if s.label == nil {
		s.label = make(map[string]string)
	}
	if s.opts.NamespaceSelectorLabel != nil {
		for k, v := range s.opts.NamespaceSelectorLabel {
			if len(v) > 0 {
				s.label[k] = v[0]
			}
		}
	} else {
		s.label["apisix.ingress.watch"] = s.namespace
	}

	// Initialize additionalGatewayGroups map
	s.additionalGateways = make(map[string]*GatewayResources)

	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: s.namespace, Labels: nsLabel})

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	Expect(err).NotTo(HaveOccurred(), "getting ready nodes")

	s.gatewayGroupID = s.CreateNewGatewayGroupWithIngress()
	s.Logf("gateway group id: %s", s.gatewayGroupID)

	s.opts.APISIXAdminAPIKey = s.GetAdminKey(s.gatewayGroupID)

	s.Logf("apisix admin api key: %s", s.opts.APISIXAdminAPIKey)

	e := utils.ParallelExecutor{}

	e.Add(func() {
		s.DeployDataplane()
		s.DeployIngress()
	})
	e.Add(s.DeployTestService)
	e.Wait()
}

func (s *API7Deployer) AfterEach() {
	defer GinkgoRecover()
	s.DeleteGatewayGroup(s.gatewayGroupID)

	if CurrentSpecReport().Failed() {
		if os.Getenv("TEST_ENV") == "CI" {
			_, _ = fmt.Fprintln(GinkgoWriter, "Dumping namespace contents")
			_, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "get", "deploy,sts,svc,pods,gatewayproxy")
			_, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "describe", "pods")
		}

		output := s.GetDeploymentLogs("apisix-ingress-controller")
		if output != "" {
			_, _ = fmt.Fprintln(GinkgoWriter, output)
		}
	}

	// Delete all additional namespaces
	for identifier := range s.additionalGateways {
		err := s.CleanupAdditionalGateway(identifier)
		Expect(err).NotTo(HaveOccurred(), "cleaning up additional gateway group")
	}

	// if the test case is successful, just delete namespace
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	Expect(err).NotTo(HaveOccurred(), "deleting namespace "+s.namespace)

	for i := len(s.finalizers) - 1; i >= 0; i-- {
		runWithRecover(s.finalizers[i])
	}

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func (s *API7Deployer) DeployDataplane() {
	svc := s.DeployGateway(framework.DataPlaneDeployOptions{
		GatewayGroupID:         s.gatewayGroupID,
		Namespace:              s.namespace,
		Name:                   "api7ee3-apisix-gateway-mtls",
		DPManagerEndpoint:      framework.DPManagerTLSEndpoint,
		SetEnv:                 true,
		SSLKey:                 framework.TestKey,
		SSLCert:                framework.TestCert,
		TLSEnabled:             true,
		ForIngressGatewayGroup: true,
		ServiceHTTPPort:        9080,
		ServiceHTTPSPort:       9443,
	})

	s.dataplaneService = svc

	err := s.newAPISIXTunnels()
	Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
}

func (s *API7Deployer) newAPISIXTunnels() error {
	serviceName := "api7ee3-apisix-gateway-mtls"
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(s.dataplaneService, s.kubectlOptions, serviceName)
	if err != nil {
		return err
	}

	s.apisixHttpTunnel = httpTunnel
	s.apisixHttpsTunnel = httpsTunnel
	return nil
}

func (s *API7Deployer) DeployIngress() {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ProviderType:   "api7ee",
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       1,
	})
}

func (s *API7Deployer) ScaleIngress(replicas int) {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ProviderType:   "api7ee",
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       replicas,
	})
}

// CreateAdditionalGateway creates a new gateway group and deploys a dataplane for it.
// It returns the gateway group ID and namespace name where the dataplane is deployed.
func (s *API7Deployer) CreateAdditionalGateway(namePrefix string) (string, *corev1.Service, error) {
	// Create a new namespace for this gateway group
	additionalNS := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())

	// Create namespace with the same labels
	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: additionalNS, Labels: nsLabel})

	// Create new kubectl options for the new namespace
	kubectlOpts := &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  additionalNS,
	}

	// Create a new gateway group
	gatewayGroupID := s.CreateNewGatewayGroupWithIngress()
	s.Logf("additional gateway group id: %s in namespace %s", gatewayGroupID, additionalNS)

	// Get the admin key for this gateway group
	adminKey := s.GetAdminKey(gatewayGroupID)
	s.Logf("additional gateway group admin api key: %s", adminKey)

	// Store gateway group info
	resources := &GatewayResources{
		Namespace:   additionalNS,
		AdminAPIKey: adminKey,
	}

	serviceName := fmt.Sprintf("api7ee3-apisix-gateway-%s", namePrefix)

	// Deploy dataplane for this gateway group
	svc := s.DeployGateway(framework.DataPlaneDeployOptions{
		GatewayGroupID:         gatewayGroupID,
		Namespace:              additionalNS,
		Name:                   serviceName,
		ServiceName:            serviceName,
		DPManagerEndpoint:      framework.DPManagerTLSEndpoint,
		SetEnv:                 true,
		SSLKey:                 framework.TestKey,
		SSLCert:                framework.TestCert,
		TLSEnabled:             true,
		ForIngressGatewayGroup: true,
		ServiceHTTPPort:        9080,
		ServiceHTTPSPort:       9443,
	})

	resources.DataplaneService = svc

	// Create tunnels for the dataplane
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(svc, kubectlOpts, serviceName)
	if err != nil {
		return "", nil, err
	}

	resources.HttpTunnel = httpTunnel
	resources.HttpsTunnel = httpsTunnel

	// Store in the map
	s.additionalGateways[gatewayGroupID] = resources

	return gatewayGroupID, svc, nil
}

// CleanupAdditionalGateway cleans up resources associated with a specific Gateway group
func (s *API7Deployer) CleanupAdditionalGateway(gatewayGroupID string) error {
	resources, exists := s.additionalGateways[gatewayGroupID]
	if !exists {
		return fmt.Errorf("gateway group %s not found", gatewayGroupID)
	}

	// Delete the gateway group
	s.DeleteGatewayGroup(gatewayGroupID)

	// Delete the namespace
	err := k8s.DeleteNamespaceE(s.t, &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  resources.Namespace,
	}, resources.Namespace)

	// Remove from the map
	delete(s.additionalGateways, gatewayGroupID)

	return err
}

func (s *API7Deployer) GetAdminEndpoint(_ ...*corev1.Service) string {
	// always return the default dashboard endpoint
	return framework.DashboardTLSEndpoint
}

func (s *API7Deployer) DefaultDataplaneResource() DataplaneResource {
	return newADCDataplaneResource(
		"api7ee",
		fmt.Sprintf("http://%s", s.GetDashboardEndpoint()),
		s.AdminKey(),
		false,
	)
}
