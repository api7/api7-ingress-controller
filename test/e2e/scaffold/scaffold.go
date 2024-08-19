// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
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
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/api7/api7-ingress/pkg/apisix"
	"github.com/api7/api7-ingress/test/e2e/framework"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	DashboardHost = "localhost"
	DashboardPort = 7080
)

var (
	tenyearsLicense = `-----BEGIN CERTIFICATE-----
MIICwjCCAaqgAwIBAgIEZl6CqjANBgkqhkiG9w0BAQsFADAeMRwwGgYDVQQDDBNB
UEk3IExpY2Vuc2UgU2VydmVyMB4XDTI0MDYwNDAyNTc0NloXDTM0MDUwNTAzMTQx
OVowFDESMBAGA1UEAxMJUHJpdmF0ZSBBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAzQOhXlXmqHFqoylKqatI0Lx+oCmF2D+1tvS4VhoSOcO9Fr48Bp6/
pLeBnmgRIAXEJryMSpljvonQJKBuUuCl83loH4Ak3payNaXInv9GAyGvzgx0Ktkb
m/8iThlnibFXGNFEjM2bRSmJa2arJB8DsZBU20n5B86ZHXYCFKzGxJig536wGyhR
FjIjD6CWOgA6d9+hybr+AhSXPLSr22isnO63TpPM2x84qePZ4u6TyiVQcvw9l5rS
9n7EKskETKBXMrLJnt2aizedBgfSxnY//XLktpCjeMjm7xH9UNQyBXiJWH+3BsXy
ThJw0mDtDpL5T1Akn0Ws4ERvIYjjWH8N3QIDAQABoxIwEDAOBgNVHQ8BAf8EBAMC
B4AwDQYJKoZIhvcNAQELBQADggEBABNg3/22QlL+z3NjsZ8qaeSpwaZSvDmn659b
AZ/JMyjym72MWc+hxSeNBKkdhuvW5Vfp3itudO4Se+UtmbdxHa2BjrjNc15kNI9E
hxPPYKs2euSRvrJltO3ZHrWcUyacdd3m26PeNIGmbQo6O2HYEpHCaqPgP4mPX24b
T1c4DJ20/vqRK7kxdRiHJuO1tgtErnkWxt3cZ0jNaNRjjtWF3toNDCzTwB8GO0y6
qZOXAx8ZONUxZLA+mPJ/+GmdtZLXot8lGccS7wS/H4lC14ClOC3BwclCWK5YisAU
/swPfbsDquG3zTFexciHBsOLefmRhRMNDuNSw5R85qnklgoKWCQ=
-----END CERTIFICATE-----
-----BEGIN LICENSE-----
ebsDHWXMfP8NYc8W8YFn0lcanxqgRhWhzzTGW7qU5kjT8xQALDVKGZB52L08Ey5qiZdQQu8ihJyA9oH5nJq_77dHq0xo9HiNfuE6g6uQ4IVOSXi8dZgTFzyyjHlwJXHL67O6c3M2bCeI6646i8eTDGPVTrMcbK-v2q0ZaZzxqNMZMu738hO3dkSwAEVp6fK298MYRviOIWdpfxPAdZu0d-csfhd0pSX7VclDUog7QRo66zCoPocMK-a3Zrp8ButyNcApmGWulG9egr1Nj3gZSNt9mU5yV-3xFtfbFl_YLJA6ll7Gk2UjamOrORoak5nLu7d0hcgCj9wxF3agN39LAQ
Gi96E_osNcD_liROu0NtvYZNrSGfDTT_yLwC7Jlsopjf5QLwPlE9yhw-FbOuHiAfCwGG0IQWLJWcROPW--8_HQKU8ujqOTsu2b5abqpx1MFyS9T35P2dqxicP9Li_XB-8dZ1jxm_Gii7uQkEUmhtCYB4EL5m9VKWwjbNmWCOfItYaHyT_87aiQHndH4wliyIAy2BpDwV-t9s7LbdHf2ZVaKFin6v0eRBqy-J7FdKhvgn-IWvm3PsUxcf2EJhXVjhBoiyIk_VCOmx1XkK9rxVqPRQOwCvpDkTtIM4vysb6nSwi5qGvYnZK1AbGqdJE0m7ydIyu3C3PT0jOrNczzvnEg
-----END LICENSE-----
`
)

type Options struct {
	Name                         string
	Kubeconfig                   string
	APISIXAdminAPIVersion        string
	APISIXConfigPath             string
	IngressAPISIXReplicas        int
	HTTPBinServicePort           int
	APISIXAdminAPIKey            string
	EnableWebhooks               bool
	APISIXPublishAddress         string
	ApisixResourceSyncInterval   string
	ApisixResourceSyncComparison string
	ApisixResourceVersion        string
	DisableStatus                bool
	IngressClass                 string
	EnableEtcdServer             bool

	NamespaceSelectorLabel   map[string][]string
	DisableNamespaceSelector bool
	DisableNamespaceLabel    bool
}
type Scaffold struct {
	*framework.Framework

	opts             *Options
	kubectlOptions   *k8s.KubectlOptions
	namespace        string
	t                testing.TestingT
	nodes            []corev1.Node
	dataplaneService *corev1.Service
	httpbinService   *corev1.Service

	finalizers []func()
	label      map[string]string

	apisixCli apisix.APISIX

	gatewaygroupid         string
	apisixHttpTunnel       *k8s.Tunnel
	apisixHttpsTunnel      *k8s.Tunnel
	apisixTCPTunnel        *k8s.Tunnel
	apisixTLSOverTCPTunnel *k8s.Tunnel
	apisixUDPTunnel        *k8s.Tunnel
	// apisixControlTunnel    *k8s.Tunnel
}

func (s *Scaffold) AdminKey() string {
	return s.opts.APISIXAdminAPIKey
}

type apisixResourceVersionInfo struct {
	V2      string
	Default string
}

var (
	createVersionedApisixResourceMap = map[string]struct{}{
		"ApisixRoute":        {},
		"ApisixConsumer":     {},
		"ApisixPluginConfig": {},
		"ApisixUpstream":     {},
	}
)

// GetKubeconfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err != nil && !os.IsNotExist(err) {
			kubeconfig = ""
		}
	}
	return kubeconfig
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	if o.Name == "" {
		o.Name = "default"
	}
	if o.IngressAPISIXReplicas <= 0 {
		o.IngressAPISIXReplicas = 1
	}
	if o.Kubeconfig == "" {
		o.Kubeconfig = GetKubeconfig()
	}
	if o.APISIXAdminAPIVersion == "" {
		adminVersion := os.Getenv("APISIX_ADMIN_API_VERSION")
		if adminVersion != "" {
			o.APISIXAdminAPIVersion = adminVersion
		} else {
			o.APISIXAdminAPIVersion = "v3"
		}
	}
	if enabled := os.Getenv("ENABLED_ETCD_SERVER"); enabled == "true" {
		o.EnableEtcdServer = true
	}

	if o.HTTPBinServicePort == 0 {
		o.HTTPBinServicePort = 80
	}
	defer GinkgoRecover()

	s := &Scaffold{
		Framework: framework.GetFramework(),
		opts:      o,
		t:         GinkgoT(),
	}

	BeforeEach(s.beforeEach)
	AfterEach(s.afterEach)

	return s
}

// NewDefaultScaffold creates a scaffold with some default options.
// apisix-version default v2
func NewDefaultScaffold() *Scaffold {
	return NewScaffold(&Options{})
}

// KillPod kill the pod which name is podName.
func (s *Scaffold) KillPod(podName string) error {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		return err
	}
	return cli.CoreV1().Pods(s.namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
}

// DefaultHTTPBackend returns the service name and service ports
// of the default http backend.
func (s *Scaffold) DefaultHTTPBackend() (string, []int32) {
	var ports []int32
	for _, p := range s.httpbinService.Spec.Ports {
		ports = append(ports, p.Port)
	}
	return s.httpbinService.Name, ports
}

// ApisixAdminServiceAndPort returns the dashboard host and port
// func (s *Scaffold) ApisixAdminServiceAndPort() (string, int32) {
// 	return "apisix-service-e2e-test", 7080
// }

// NewAPISIXClient creates the default HTTP client.
func (s *Scaffold) NewAPISIXClient() *httpexpect.Expect {
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

// GetAPISIXHTTPSEndpoint get apisix https endpoint from tunnel map
func (s *Scaffold) GetAPISIXHTTPSEndpoint() string {
	return s.apisixHttpsTunnel.Endpoint()
}

// NewAPISIXClientWithTCPProxy creates the HTTP client but with the TCP proxy of APISIX.
func (s *Scaffold) NewAPISIXClientWithTCPProxy() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixTCPTunnel.Endpoint(),
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

func (s *Scaffold) DNSResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, "udp", s.apisixUDPTunnel.Endpoint())
		},
	}
}

func (s *Scaffold) DialTLSOverTcp(serverName string) (*tls.Conn, error) {
	return tls.Dial("tcp", s.apisixTLSOverTCPTunnel.Endpoint(), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	})
}

func (s *Scaffold) UpdateNamespace(ns string) {
	s.kubectlOptions.Namespace = ns
}

// NewAPISIXHttpsClient creates the default HTTPS client.
func (s *Scaffold) NewAPISIXHttpsClient(host string) *httpexpect.Expect {
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

// NewAPISIXHttpsClientWithCertificates creates the default HTTPS client with giving trusted CA and client certs.
func (s *Scaffold) NewAPISIXHttpsClientWithCertificates(host string, insecure bool, ca *x509.CertPool, certs []tls.Certificate) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   s.apisixHttpsTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
					ServerName:         host,
					RootCAs:            ca,
					Certificates:       certs,
				},
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

// APISIXGatewayServiceEndpoint returns the apisix http gateway endpoint.
func (s *Scaffold) APISIXGatewayServiceEndpoint() string {
	return s.apisixHttpTunnel.Endpoint()
}

// RestartAPISIXDeploy delete apisix pod and wait new pod be ready
func (s *Scaffold) RestartAPISIXDeploy() {
	s.shutdownApisixTunnel()
	pods, err := k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=apisix-deployment-e2e-test",
	})
	assert.NoError(s.t, err, "list apisix pod")
	for _, pod := range pods {
		err = s.KillPod(pod.Name)
		assert.NoError(s.t, err, "killing apisix pod")
	}
	err = s.waitAllAPISIXPodsAvailable()
	assert.NoError(s.t, err, "waiting for new apisix instance ready")
	err = s.newAPISIXTunnels()
	assert.NoError(s.t, err, "renew apisix tunnels")
}

func (s *Scaffold) RestartIngressControllerDeploy() {
	pods, err := k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	})
	assert.NoError(s.t, err, "list ingress-controller pod")
	for _, pod := range pods {
		err = s.KillPod(pod.Name)
		assert.NoError(s.t, err, "killing ingress-controller pod")
	}
}

func (s *Scaffold) beforeEach() {
	var err error
	s.UploadLicense()
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
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

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	assert.Nil(s.t, err, "querying ready nodes")

	s.gatewaygroupid = s.CreateNewGatewayGroupWithIngress()
	s.Logger.Logf(s.t, "gateway group id: %s", s.gatewaygroupid)

	s.opts.APISIXAdminAPIKey, err = s.GetAPIKey()
	assert.Nil(s.t, err, "getting api key")

	s.Logger.Logf(s.t, "apisix admin api key: %s", s.opts.APISIXAdminAPIKey)

	s.DeployDataplaneWithIngress()
	s.DeployTestService()
	s.initDataPlaneClient()
}

func (s *Scaffold) initDataPlaneClient() {
	var err error
	s.apisixCli, err = apisix.NewClient()
	assert.Nil(s.t, err, "creating apisix client")

	url := fmt.Sprintf("http://%s/apisix/admin", s.GetDashboardEndpoint())

	s.Logger.Logf(s.t, "apisix admin: %s", url)

	err = s.apisixCli.AddCluster(context.Background(), &apisix.ClusterOptions{
		Name:     "default",
		BaseURL:  url,
		AdminKey: s.AdminKey(),
	})
	assert.Nil(s.t, err, "adding cluster")
}

func (s *Scaffold) DefaultDataplaneResource() apisix.Cluster {
	return s.apisixCli.Cluster("default")
}

func (s *Scaffold) DataPlaneClient() apisix.APISIX {
	return s.apisixCli
}

func (s *Scaffold) DeployTestService() {
	var err error

	s.httpbinService, err = s.newHTTPBIN()
	assert.Nil(s.t, err, "initializing httpbin")
	s.EnsureNumEndpointsReady(s.t, s.httpbinService.Name, 1)
}

func (s *Scaffold) afterEach() {
	s.DeleteGatewayGroup()
	defer GinkgoRecover()

	if CurrentSpecReport().Failed() {
		// dump and delete related resource
		env := os.Getenv("E2E_ENV")
		if env == "ci" {
			_, _ = fmt.Fprintln(GinkgoWriter, "Dumping namespace contents")
			output, _ := k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "get", "deploy,sts,svc,pods")
			if output != "" {
				_, _ = fmt.Fprintln(GinkgoWriter, output)
			}
			output, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "describe", "pods")
			if output != "" {
				_, _ = fmt.Fprintln(GinkgoWriter, output)
			}
			// Get the logs of apisix
			output = s.GetDeploymentLogs("apisix-deployment-e2e-test")
			if output != "" {
				_, _ = fmt.Fprintln(GinkgoWriter, output)
			}
			// Get the logs of ingress
			output = s.GetDeploymentLogs("ingress-apisix-controller-deployment-e2e-test")
			if output != "" {
				_, _ = fmt.Fprintln(GinkgoWriter, output)
			}
			if s.opts.EnableWebhooks {
				output, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "get", "validatingwebhookconfigurations", "-o", "yaml")
				if output != "" {
					_, _ = fmt.Fprintln(GinkgoWriter, output)
				}
			}
		}
		if env != "debug" {
			err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
			assert.Nilf(GinkgoT(), err, "deleting namespace %s", s.namespace)
		}
	} else {
		// if the test case is successful, just delete namespace
		err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
		assert.Nilf(GinkgoT(), err, "deleting namespace %s", s.namespace)
	}

	for _, f := range s.finalizers {
		runWithRecover(f)
	}

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func runWithRecover(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if ok {
			// just ignore already closed channel
			if strings.Contains(err.Error(), "close of closed channel") {
				return
			}
		}
		panic(r)
	}()
	f()
}

func (s *Scaffold) GetDeploymentLogs(name string) string {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		assert.Nilf(GinkgoT(), err, "get client error: %s", err.Error())
	}
	pods, err := cli.CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=" + name,
	})
	if err != nil {
		return ""
	}
	var buf strings.Builder
	for _, pod := range pods.Items {
		buf.WriteString(fmt.Sprintf("=== pod: %s ===\n", pod.Name))
		logs, err := cli.CoreV1().RESTClient().Get().
			Resource("pods").
			Namespace(s.namespace).
			Name(pod.Name).SubResource("log").
			Param("container", name).
			Do(context.TODO()).
			Raw()
		if err == nil {
			buf.Write(logs)
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (s *Scaffold) addFinalizers(f func()) {
	s.finalizers = append(s.finalizers, f)
}

func (s *Scaffold) renderConfig(path string, config any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	t := template.Must(template.New(path).Parse(string(data)))
	if err := t.Execute(&buf, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FormatRegistry replace default registry to custom registry if exist
func (s *Scaffold) FormatRegistry(workloadTemplate string) string {
	customRegistry, isExist := os.LookupEnv("REGISTRY")
	if isExist {
		return strings.Replace(workloadTemplate, "127.0.0.1:5000", customRegistry, -1)
	} else {
		return workloadTemplate
	}
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

func (s *Scaffold) DeleteResource(resourceType, name string) error {
	return k8s.RunKubectlE(s.t, s.kubectlOptions, "delete", resourceType, name)
}

func (s *Scaffold) NamespaceSelectorLabelStrings() []string {
	var labels []string
	if s.opts.NamespaceSelectorLabel != nil {
		for k, v := range s.opts.NamespaceSelectorLabel {
			for _, v0 := range v {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v0))
			}
		}
	} else {
		for k, v := range s.label {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return labels
}

func (s *Scaffold) NamespaceSelectorLabel() map[string][]string {
	return s.opts.NamespaceSelectorLabel
}
func (s *Scaffold) labelSelector(label string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: label,
	}
}
