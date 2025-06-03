package scaffold

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
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
	s.additionalGatewayGroups = make(map[string]*GatewayGroupResources)

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
		s.initDataPlaneClient()
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
	for _, resources := range s.additionalGatewayGroups {
		err := s.CleanupAdditionalGatewayGroup(resources.GatewayGroupID)
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
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       1,
	})
}

func (s *API7Deployer) ScaleIngress(replicas int) {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       replicas,
	})
}

func (s *API7Deployer) initDataPlaneClient() {
	var err error
	s.apisixCli, err = dashboard.NewClient()
	Expect(err).NotTo(HaveOccurred(), "creating apisix client")

	url := fmt.Sprintf("http://%s/apisix/admin", s.GetDashboardEndpoint())

	s.Logf("apisix admin: %s", url)

	err = s.apisixCli.AddCluster(context.Background(), &dashboard.ClusterOptions{
		Name:           "default",
		ControllerName: s.opts.ControllerName,
		Labels:         map[string]string{"k8s/controller-name": s.opts.ControllerName},
		BaseURL:        url,
		AdminKey:       s.AdminKey(),
	})
	Expect(err).NotTo(HaveOccurred(), "adding cluster")

	httpsURL := fmt.Sprintf("https://%s/apisix/admin", s.GetDashboardEndpointHTTPS())
	err = s.apisixCli.AddCluster(context.Background(), &dashboard.ClusterOptions{
		Name:          "default-https",
		BaseURL:       httpsURL,
		AdminKey:      s.AdminKey(),
		SkipTLSVerify: true,
	})
	Expect(err).NotTo(HaveOccurred(), "adding cluster")
}
