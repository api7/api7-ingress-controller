// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package scaffold

import (
	"fmt"
	"os"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/pkg/utils"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

type API7Deployer struct {
	*Scaffold

	gatewayGroupID string
}

func NewAPI7Deployer(s *Scaffold) Deployer {
	return &API7Deployer{
		Scaffold: s,
	}
}

func (s *API7Deployer) BeforeEach() {
	s.runtimeOpts = s.opts
	var err error
	s.UploadLicense()
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.runtimeOpts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.runtimeOpts.Kubeconfig,
		Namespace:  s.namespace,
	}
	if s.runtimeOpts.ControllerName == "" {
		s.runtimeOpts.ControllerName = fmt.Sprintf("%s/%s", DefaultControllerName, s.namespace)
	}
	s.finalizers = nil

	// Initialize additionalGatewayGroups map
	s.additionalGateways = make(map[string]*GatewayResources)

	k8s.CreateNamespace(s.t, s.kubectlOptions, s.namespace)

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	Expect(err).NotTo(HaveOccurred(), "getting ready nodes")

	s.gatewayGroupID = s.CreateNewGatewayGroupWithIngress()
	s.Logf("gateway group id: %s", s.gatewayGroupID)

	s.runtimeOpts.APISIXAdminAPIKey = s.GetAdminKey(s.gatewayGroupID)

	s.Logf("apisix admin api key: %s", s.runtimeOpts.APISIXAdminAPIKey)

	e := utils.ParallelExecutor{}

	e.Add(func() {
		defer GinkgoRecover()
		s.DeployDataplane(DeployDataplaneOptions{})
		s.DeployIngress()
	})
	e.Add(func() {
		defer GinkgoRecover()
		s.DeployTestService()
	})
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

	for i := len(s.finalizers) - 1; i >= 0; i-- {
		runWithRecover(s.finalizers[i])
	}

	// if the test case is successful, just delete namespace
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	Expect(err).NotTo(HaveOccurred(), "deleting namespace "+s.namespace)

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func (s *API7Deployer) DeployDataplane(deployOpts DeployDataplaneOptions) {
	opts := framework.API7DeployOptions{
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
		Replicas:               ptr.To(1),
	}
	if deployOpts.Namespace != "" {
		opts.Namespace = deployOpts.Namespace
	}
	if deployOpts.ServiceType != "" {
		opts.ServiceType = deployOpts.ServiceType
	}
	if deployOpts.ServiceHTTPPort != 0 {
		opts.ServiceHTTPPort = deployOpts.ServiceHTTPPort
	}
	if deployOpts.ServiceHTTPSPort != 0 {
		opts.ServiceHTTPSPort = deployOpts.ServiceHTTPSPort
	}
	if deployOpts.Replicas != nil {
		opts.Replicas = deployOpts.Replicas
	}
	if opts.Replicas != nil && *opts.Replicas == 0 {
		deployOpts.SkipCreateTunnels = true
	}

	if s.apisixTunnels != nil {
		s.apisixTunnels.Close()
	}

	svc := s.DeployGateway(&opts)
	s.dataplaneService = svc

	if !deployOpts.SkipCreateTunnels {
		err := s.newAPISIXTunnels(opts.ServiceName)
		Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
	}
}

func (s *API7Deployer) ScaleDataplane(replicas int) {
	s.DeployDataplane(DeployDataplaneOptions{
		Replicas: ptr.To(replicas),
	})
}

func (s *API7Deployer) newAPISIXTunnels(serviceName string) error {
	apisixTunnels, err := s.createDataplaneTunnels(s.dataplaneService, s.kubectlOptions, serviceName)
	if err != nil {
		return err
	}

	s.apisixTunnels = apisixTunnels
	return nil
}

func (s *API7Deployer) DeployIngress() {
	if s.runtimeOpts.EnableWebhook {
		err := s.SetupWebhookResources()
		Expect(err).NotTo(HaveOccurred(), "setting up webhook resources")
	}

	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ProviderType:   "api7ee",
		ControllerName: s.runtimeOpts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       ptr.To(1),
	})
}

func (s *API7Deployer) ScaleIngress(replicas int) {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ProviderType:   "api7ee",
		ControllerName: s.runtimeOpts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       ptr.To(replicas),
	})
}

// CreateAdditionalGateway creates a new gateway group and deploys a dataplane for it.
// It returns the gateway group ID and namespace name where the dataplane is deployed.
func (s *API7Deployer) CreateAdditionalGateway(namePrefix string) (string, *corev1.Service, error) {
	// Create a new namespace for this gateway group
	additionalNS := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())

	k8s.CreateNamespace(s.t, s.kubectlOptions, additionalNS)

	// Create new kubectl options for the new namespace
	kubectlOpts := &k8s.KubectlOptions{
		ConfigPath: s.runtimeOpts.Kubeconfig,
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
	svc := s.DeployGateway(&framework.API7DeployOptions{
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
	tunnels, err := s.createDataplaneTunnels(svc, kubectlOpts, serviceName)
	if err != nil {
		return "", nil, err
	}

	resources.Tunnels = tunnels

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
		ConfigPath: s.runtimeOpts.Kubeconfig,
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

func (s *API7Deployer) Name() string {
	return "api7ee"
}
