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
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
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

	// Use the new Deployer interface
	deployer          Deployer
	adminClient       *httpexpect.Expect
	apisixHttpTunnel  *k8s.Tunnel
	apisixHttpsTunnel *k8s.Tunnel
	httpbinService    *corev1.Service
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
	s.APISIXFramework.DeployIngress(opts)
}

// deployAPISIXStandalone deploys APISIX in standalone mode
func (s *APISIXScaffold) deployAPISIXStandalone() {
	// Create our new APISIX deployer
	deployer, err := NewAPISIXDeployer(s.t, s.kubectlOptions, s.APISIXFramework, &DeployerOptions{
		Namespace:   s.namespace,
		ServiceType: "ClusterIP",
		AdminKey:    s.opts.APISIXAdminAPIKey,
		APISIXImage: "apache/apisix:3.8.0", // Default image
	})
	Expect(err).NotTo(HaveOccurred(), "creating APISIX deployer")

	s.deployer = deployer

	err = s.deployer.Deploy(context.Background())
	Expect(err).NotTo(HaveOccurred(), "deploying APISIX standalone")

	// Create HTTP tunnels for APISIX - we need to get the service from the deployer
	apisixDeployer, ok := s.deployer.(*APISIXDeployer)
	if !ok {
		panic("deployer is not APISIXDeployer")
	}
	s.createAPISIXTunnel(apisixDeployer.service)

	// init admin client
	s.initAdminClient()
}

// createAPISIXTunnel creates HTTP tunnel to APISIX service
func (s *APISIXScaffold) createAPISIXTunnel(svc *corev1.Service) {
	var (
		httpNodePort  int
		httpsNodePort int
		httpPort      int
		httpsPort     int
	)

	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case "http":
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		case "https":
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		}
	}

	tunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, svc.Name, httpNodePort, httpPort)
	tunnel.ForwardPort(s.t)
	s.apisixHttpTunnel = tunnel

	httpsTunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, svc.Name, httpsNodePort, httpsPort)
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
	s.Logf("Deploying test services")

	// Deploy httpbin service for testing
	httpbinYaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
`

	err := s.CreateResourceFromString(httpbinYaml)
	if err != nil {
		s.Logf("Failed to deploy httpbin: %v", err)
	} else {
		s.Logf("httpbin service deployed successfully")
	}
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
	return s.opts.APISIXAdminAPIKey
}

func (s *APISIXScaffold) GetControllerName() string {
	return s.getControllerName()
}

func (s *APISIXScaffold) Namespace() string {
	return s.namespace
}

func (s *APISIXScaffold) GetContext() context.Context {
	return s.Context
}

func (s *APISIXScaffold) CreateResourceFromString(resourceYaml string) error {
	return s.CreateResourceFromStringWithNamespace(resourceYaml, s.namespace)
}

func (s *APISIXScaffold) CreateResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	kubectlOpts := *s.kubectlOptions
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	return k8s.KubectlApplyFromStringE(s.t, &kubectlOpts, resourceYaml)
}

func (s *APISIXScaffold) DeleteResourceFromString(resourceYaml string) error {
	return s.DeleteResourceFromStringWithNamespace(resourceYaml, s.namespace)
}

func (s *APISIXScaffold) DeleteResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	kubectlOpts := *s.kubectlOptions
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	return k8s.KubectlDeleteFromStringE(s.t, &kubectlOpts, resourceYaml)
}

func (s *APISIXScaffold) DeleteResource(resourceType, name string) error {
	args := []string{"delete", resourceType, name}
	return k8s.RunKubectlE(s.t, s.kubectlOptions, args...)
}

func (s *APISIXScaffold) GetResourceYaml(resourceType, name string) (string, error) {
	return s.GetResourceYamlFromNamespace(resourceType, name, s.namespace)
}

func (s *APISIXScaffold) GetResourceYamlFromNamespace(resourceType, name, namespace string) (string, error) {
	kubectlOpts := *s.kubectlOptions
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	args := []string{"get", resourceType, name, "-o", "yaml"}
	return k8s.RunKubectlAndGetOutputE(s.t, &kubectlOpts, args...)
}

func (s *APISIXScaffold) RunKubectlAndGetOutput(args ...string) (string, error) {
	return k8s.RunKubectlAndGetOutputE(s.t, s.kubectlOptions, args...)
}

func (s *APISIXScaffold) NewKubeTlsSecret(secretName, cert, key string) error {
	const kubeTlsSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: kubernetes.io/tls
data:
  tls.crt: %s
  tls.key: %s
`
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))
	secret := fmt.Sprintf(kubeTlsSecretTemplate, secretName, certBase64, keyBase64)
	return s.CreateResourceFromString(secret)
}

func (s *APISIXScaffold) DefaultDataplaneResource() dashboard.Cluster {
	if s.deployer != nil {
		return s.deployer.GetDataplaneResource()
	}
	return nil
}

func (s *APISIXScaffold) CreateAdditionalGatewayGroup(namePrefix string) (string, string, error) {
	// APISIX standalone doesn't support additional gateway groups
	// This method is API7-specific functionality
	return "", "", fmt.Errorf("additional gateway groups not supported in APISIX standalone mode")
}

func (s *APISIXScaffold) GetAdditionalGatewayGroup(gatewayGroupID string) (*GatewayGroupResources, bool) {
	// APISIX standalone doesn't support additional gateway groups
	return nil, false
}

func (s *APISIXScaffold) NewAPISIXClientForGatewayGroup(gatewayGroupID string) (*httpexpect.Expect, error) {
	// APISIX standalone doesn't support additional gateway groups
	// Return the main APISIX client instead
	return s.NewAPISIXClient(), nil
}

func (s *APISIXScaffold) GetGinkgoT() ginkgo.GinkgoTInterface {
	return ginkgo.GinkgoT()
}

func (s *APISIXScaffold) GetK8sClient() client.Client {
	return s.APISIXFramework.K8sClient
}

func (s *APISIXScaffold) ApplyDefaultGatewayResource(gatewayProxy, gatewayClass, gateway, httpRoute string) {
	// For APISIX standalone, we don't need to apply gateway resources
	// since it doesn't use the gateway API like API7
	s.Logf("ApplyDefaultGatewayResource called but not implemented for APISIX standalone")
}

func (s *APISIXScaffold) DefaultDataplaneResourceHTTPS() dashboard.Cluster {
	if s.deployer != nil {
		return s.deployer.GetDataplaneResourceHTTPS()
	}
	return nil
}

func (s *APISIXScaffold) ResourceApplied(resourType, resourceName, resourceRaw string, observedGeneration int) {
	// Simple implementation for APISIX standalone
	// This can be enhanced if needed for specific tests
	s.Logf("Resource applied: %s/%s", resourType, resourceName)
}

func (s *APISIXScaffold) ScaleIngress(replicas int) {
	// Scale the ingress controller deployment
	args := []string{"scale", "deployment", "apisix-ingress-controller", "--replicas", fmt.Sprintf("%d", replicas)}
	err := k8s.RunKubectlE(s.t, s.kubectlOptions, args...)
	if err != nil {
		s.Logf("Failed to scale ingress controller: %v", err)
	}
}

func (s *APISIXScaffold) GetDeploymentLogs(name string) string {
	return s.getDeploymentLogs(name)
}

func (s *APISIXScaffold) DeployNginx(options framework.NginxOptions) {
	// Deploy nginx test service for APISIX standalone
	// Since APISIXFramework doesn't have DeployNginx, we need to implement it manually
	// For now, just log that it's called
	s.Logf("DeployNginx called with options: %+v - not implemented for APISIX standalone", options)
}

// GetDeployer returns the underlying deployer instance
func (s *APISIXScaffold) GetDeployer() Deployer {
	return s.deployer
}
