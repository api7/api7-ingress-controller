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
	"context"
	"crypto/tls"
	"fmt"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// APISIXScaffold implements TestScaffold for APISIX standalone
type APISIXScaffold struct {
	*framework.APISIXFramework

	opts           *Options
	kubectlOptions *k8s.KubectlOptions
	namespace      string
	t              testing.TestingT
	nodes          []corev1.Node

	finalizers []func()

	deployer          *framework.APISIXDeployer
	adminClient       *httpexpect.Expect
	apisixHttpTunnel  *k8s.Tunnel
	apisixHttpsTunnel *k8s.Tunnel
	httpbinService    *corev1.Service
}

func (s *APISIXScaffold) ResourceApplied(resourType, resourceName, resourceRaw string, observedGeneration int) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) ScaleIngress(i int) {
	//TODO implement me
	panic("implement me")
}

// NewAPISIXScaffold creates a new APISIX scaffold
func NewAPISIXScaffold(opts *Options) *APISIXScaffold {
	if opts == nil {
		opts = &Options{}
	}

	// Set default values
	if opts.Name == "" {
		opts.Name = "default"
	}
	if opts.IngressAPISIXReplicas <= 0 {
		opts.IngressAPISIXReplicas = 1
	}
	if opts.HTTPBinServicePort == 0 {
		opts.HTTPBinServicePort = 80
	}

	s := &APISIXScaffold{
		APISIXFramework: framework.GetAPISIXFramework(),
		opts:            opts,
		t:               GinkgoT(),
	}

	BeforeEach(s.BeforeEach)
	AfterEach(s.AfterEach)

	return s
}

// BeforeEach sets up the test environment for each test
func (s *APISIXScaffold) BeforeEach() {
	var err error

	s.namespace = fmt.Sprintf("apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
	}

	s.finalizers = nil

	// Create test namespace
	k8s.CreateNamespace(s.t, s.kubectlOptions, s.namespace)

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	Expect(err).NotTo(HaveOccurred(), "getting ready nodes")

	// Deploy APISIX standalone
	s.deployAPISIXStandalone()

	// Deploy ingress controller
	s.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.getControllerName(),
		Namespace:      s.namespace,
		Replicas:       s.opts.IngressAPISIXReplicas,
	})

	// Deploy test services
	s.deployTestServices()
}

// AfterEach cleans up after each test
func (s *APISIXScaffold) AfterEach() {
	defer GinkgoRecover()

	if CurrentSpecReport().Failed() {
		s.Logf("Test failed, dumping logs")
		output := s.getDeploymentLogs("apisix-ingress-controller")
		if output != "" {
			_, _ = fmt.Fprintln(GinkgoWriter, output)
		}
	}

	// Clean up namespace
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	Expect(err).NotTo(HaveOccurred(), "deleting namespace "+s.namespace)

	// Run finalizers
	for i := len(s.finalizers) - 1; i >= 0; i-- {
		s.runWithRecover(s.finalizers[i])
	}

	// Wait to prevent overwhelming the worker node
	time.Sleep(3 * time.Second)
}

// NewAPISIXClient returns HTTP client for APISIX
func (s *APISIXScaffold) NewAPISIXClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixHttpTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

// NewAPISIXHttpsClient returns HTTPS client for APISIX
func (s *APISIXScaffold) NewAPISIXHttpsClient(host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   s.apisixHttpsTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// accept any certificate; for testing only!
					InsecureSkipVerify: true,
					ServerName:         host,
				},
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

// DeployIngress deploys the ingress controller
func (s *APISIXScaffold) DeployIngress(opts framework.IngressDeployOpts) {
	s.DeployIngress(opts)
}

// deployAPISIXStandalone deploys APISIX in standalone mode
func (s *APISIXScaffold) deployAPISIXStandalone() {
	deployOpts := &framework.APISIXDeployOptions{
		Namespace:   s.namespace,
		ServiceType: "ClusterIP",
	}

	s.deployer = framework.NewAPISIXDeployer(s.t, s.kubectlOptions, deployOpts)

	err := s.deployer.Deploy(context.Background())
	Expect(err).NotTo(HaveOccurred(), "deploying APISIX standalone")

	// Create HTTP tunnel for APISIX
	s.createAPISIXTunnel()
	// init admin client
	s.initAdminClient()
}

// createAPISIXTunnel creates HTTP tunnel to APISIX service
func (s *APISIXScaffold) createAPISIXTunnel() {
	tunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix", 0, 80)
	tunnel.ForwardPort(s.t)
	s.apisixHttpTunnel = tunnel

	httpsTunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix", 0, 443)
	httpsTunnel.ForwardPort(s.t)
	s.apisixHttpsTunnel = httpsTunnel

	s.addFinalizers(func() {
		tunnel.Close()
		httpsTunnel.Close()
	})
}

// initAdminClient initializes the APISIX admin client
func (s *APISIXScaffold) initAdminClient() {
	adminTunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix", 0, 9180)
	adminTunnel.ForwardPort(s.t)

	s.addFinalizers(func() {
		adminTunnel.Close()
	})

	adminEndpoint := fmt.Sprintf("http://%s", adminTunnel.Endpoint())
	s.adminClient = httpexpect.Default(GinkgoT(), adminEndpoint)
}

// deployTestServices deploys test services like httpbin
func (s *APISIXScaffold) deployTestServices() {
	// TODO: Implement httpbin deployment
	s.Logf("Deploying test services")
}

// getControllerName returns the controller name
func (s *APISIXScaffold) getControllerName() string {
	if s.opts.ControllerName == "" {
		return fmt.Sprintf("%s/%d", DefaultControllerName, time.Now().Nanosecond())
	}
	return s.opts.ControllerName
}

// addFinalizers adds cleanup functions
func (s *APISIXScaffold) addFinalizers(f func()) {
	s.finalizers = append(s.finalizers, f)
}

// runWithRecover runs a function with panic recovery
func (s *APISIXScaffold) runWithRecover(f func()) {
	defer func() {
		if r := recover(); r != nil {
			s.Logf("Recovered from panic in finalizer: %v", r)
		}
	}()
	f()
}

// getDeploymentLogs gets logs from a deployment
func (s *APISIXScaffold) getDeploymentLogs(name string) string {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		return ""
	}

	pods, err := cli.CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=" + name,
	})
	if err != nil {
		return ""
	}

	var logs string
	for _, pod := range pods.Items {
		logs += fmt.Sprintf("=== pod: %s ===\n", pod.Name)
		podLogs, err := cli.CoreV1().RESTClient().Get().
			Resource("pods").
			Namespace(s.namespace).
			Name(pod.Name).SubResource("log").
			Do(context.TODO()).
			Raw()
		if err == nil {
			logs += string(podLogs)
		}
		logs += "\n"
	}
	return logs
}

func (s *APISIXScaffold) AdminKey() string {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetControllerName() string {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) Namespace() string {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) Context() context.Context {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) CreateResourceFromString(resourceYaml string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) CreateResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) DeleteResourceFromString(resourceYaml string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) DeleteResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) DeleteResource(resourceType, name string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetResourceYaml(resourceType, name string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetResourceYamlFromNamespace(resourceType, name, namespace string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) RunKubectlAndGetOutput(args ...string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) NewKubeTlsSecret(secretName, cert, key string) error {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) DefaultDataplaneResource() dashboard.Cluster {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) CreateAdditionalGatewayGroup(namePrefix string) (string, string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetAdditionalGatewayGroup(gatewayGroupID string) (*GatewayGroupResources, bool) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) NewAPISIXClientForGatewayGroup(gatewayGroupID string) (*httpexpect.Expect, error) {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetContext() context.Context {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetGinkgoT() GinkgoTInterface {
	//TODO implement me
	panic("implement me")
}

func (s *APISIXScaffold) GetK8sClient() client.Client {
	//TODO implement me
	panic("implement me")
}
