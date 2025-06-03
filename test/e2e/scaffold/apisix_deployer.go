package scaffold

import (
	"bytes"
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

type APISIXDeployOptions struct {
	Namespace string
	AdminKey  string

	ServiceName      string
	ServiceType      string
	ServiceHTTPPort  int
	ServiceHTTPSPort int
}

type APISIXDeployer struct {
	*Scaffold
}

func NewAPISIXDeployer(s *Scaffold) *APISIXDeployer {
	return &APISIXDeployer{
		Scaffold: s,
	}
}

func (s *APISIXDeployer) BeforeEach() {
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

	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: s.namespace, Labels: nsLabel})

	if s.opts.APISIXAdminAPIKey == "" {
		s.opts.APISIXAdminAPIKey = getEnvOrDefault("APISIX_ADMIN_KEY", "edd1c9f034335f136f87ad84b625c8f1")
	}

	s.Logf("apisix admin api key: %s", s.opts.APISIXAdminAPIKey)

	e := utils.ParallelExecutor{}

	e.Add(func() {
		s.DeployDataplane()
		s.DeployIngress()
	})
	e.Add(s.DeployTestService)
	e.Wait()
}

func (s *APISIXDeployer) AfterEach() {
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

func (s *APISIXDeployer) DeployDataplane() {
	opts := APISIXDeployOptions{
		Namespace:        s.namespace,
		AdminKey:         s.opts.APISIXAdminAPIKey,
		ServiceHTTPPort:  9080,
		ServiceHTTPSPort: 9443,
	}
	svc := s.deployDataplane(&opts)
	s.dataplaneService = svc

	err := s.newAPISIXTunnels(opts.ServiceName)
	Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
}

func (s *APISIXDeployer) newAPISIXTunnels(serviceName string) error {
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(s.dataplaneService, s.kubectlOptions, serviceName)
	if err != nil {
		return err
	}

	s.apisixHttpTunnel = httpTunnel
	s.apisixHttpsTunnel = httpsTunnel
	return nil
}

func (s *APISIXDeployer) deployDataplane(opts *APISIXDeployOptions) *corev1.Service {
	if opts.ServiceName == "" {
		opts.ServiceName = "apisix-standalone"
	}

	if opts.ServiceHTTPPort == 0 {
		opts.ServiceHTTPPort = 80
	}

	if opts.ServiceHTTPSPort == 0 {
		opts.ServiceHTTPSPort = 443
	}

	buf := bytes.NewBuffer(nil)
	err := framework.APISIXStandaloneTpl.Execute(buf, opts)
	Expect(err).ToNot(HaveOccurred(), "executing template")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)
	k8s.KubectlApplyFromString(s.GinkgoT, kubectlOpts, buf.String())

	err = framework.WaitPodsAvailable(s.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=apisix",
	})
	Expect(err).ToNot(HaveOccurred(), "waiting for gateway pod ready")

	Eventually(func() bool {
		svc, err := k8s.GetServiceE(s.GinkgoT, kubectlOpts, opts.ServiceName)
		if err != nil {
			s.Logf("failed to get service %s: %v", opts.ServiceName, err)
			return false
		}
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			return len(svc.Status.LoadBalancer.Ingress) > 0
		}
		return true
	}, "20s", "4s").Should(BeTrue(), "waiting for LoadBalancer IP")

	svc, err := k8s.GetServiceE(s.GinkgoT, kubectlOpts, opts.ServiceName)
	Expect(err).ToNot(HaveOccurred(), "failed to get service %s: %v", opts.ServiceName, err)
	return svc
}

func (s *APISIXDeployer) DeployIngress() {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       1,
	})
}

func (s *APISIXDeployer) ScaleIngress(replicas int) {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		Namespace:      s.namespace,
		Replicas:       replicas,
	})
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

//nolint:unused
func (s *APISIXDeployer) createAdminTunnel(
	svc *corev1.Service,
	kubectlOpts *k8s.KubectlOptions,
	serviceName string,
) (*k8s.Tunnel, error) {
	var (
		adminNodePort int
		adminPort     int
	)

	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case "admin":
			adminNodePort = int(port.NodePort)
			adminPort = int(port.Port)
		}
	}

	adminTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		adminNodePort, adminPort)

	if err := adminTunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	s.addFinalizers(adminTunnel.Close)

	return adminTunnel, nil
}
