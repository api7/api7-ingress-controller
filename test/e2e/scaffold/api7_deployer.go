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
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// API7Deployer implements Deployer interface for API7 enterprise version
type API7Deployer struct {
	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	framework   *framework.Framework
	opts        *DeployerOptions

	// API7-specific resources
	apisixCli   dashboard.Dashboard
	httpTunnel  *k8s.Tunnel
	httpsTunnel *k8s.Tunnel
}

// NewAPI7Deployer creates a new API7 deployer
func NewAPI7Deployer(t testing.TestingT, kubectlOpts *k8s.KubectlOptions, framework *framework.Framework, opts *DeployerOptions) (Deployer, error) {
	if opts == nil {
		return nil, fmt.Errorf("deployer options cannot be nil")
	}

	return &API7Deployer{
		t:           t,
		kubectlOpts: kubectlOpts,
		framework:   framework,
		opts:        opts,
	}, nil
}

// Deploy deploys API7 dashboard and gateway
func (d *API7Deployer) Deploy(ctx context.Context) error {
	// Deploy API7 dashboard and gateway
	// This will use the existing framework logic
	d.framework.DeployComponents()

	// Initialize data plane client
	return d.initDataPlaneClient()
}

// Cleanup cleans up API7 resources
func (d *API7Deployer) Cleanup(ctx context.Context) error {
	// Cleanup logic for API7 resources
	// This should clean up dashboard, gateway, etc.
	return nil
}

// GetHTTPClient returns HTTP client for API7 gateway
func (d *API7Deployer) GetHTTPClient() *httpexpect.Expect {
	if d.httpTunnel == nil {
		// Create tunnel if not exists
		d.createTunnels()
	}

	u := url.URL{
		Scheme: "http",
		Host:   d.httpTunnel.Endpoint(),
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

// GetHTTPSClient returns HTTPS client for API7 gateway
func (d *API7Deployer) GetHTTPSClient(host string) *httpexpect.Expect {
	if d.httpsTunnel == nil {
		// Create tunnel if not exists
		d.createTunnels()
	}

	return d.newHTTPSClientWithCertificates(host, true, nil, nil)
}

// GetAdminClient returns admin client for API7 dashboard
func (d *API7Deployer) GetAdminClient() *httpexpect.Expect {
	// TODO: This needs to be adapted to return httpexpect/v2.Expect
	// For now, we'll return a new client pointing to dashboard
	u := url.URL{
		Scheme: "http",
		Host:   d.framework.GetDashboardEndpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

// GetAdminKey returns admin key for API7
func (d *API7Deployer) GetAdminKey() string {
	return d.opts.AdminKey
}

// GetDataplaneResource returns dashboard cluster for admin operations
func (d *API7Deployer) GetDataplaneResource() dashboard.Cluster {
	return d.apisixCli.Cluster("default")
}

// GetDataplaneResourceHTTPS returns HTTPS dashboard cluster
func (d *API7Deployer) GetDataplaneResourceHTTPS() dashboard.Cluster {
	return d.apisixCli.Cluster("default-https")
}

// GetMode returns the deployment mode
func (d *API7Deployer) GetMode() DeployMode {
	return DeployModeAPI7
}

// initDataPlaneClient initializes the dashboard client
func (d *API7Deployer) initDataPlaneClient() error {
	var err error
	d.apisixCli, err = dashboard.NewClient()
	if err != nil {
		return fmt.Errorf("creating apisix client: %w", err)
	}

	url := fmt.Sprintf("http://%s/apisix/admin", d.framework.GetDashboardEndpoint())

	err = d.apisixCli.AddCluster(context.Background(), &dashboard.ClusterOptions{
		Name:           "default",
		ControllerName: d.opts.ControllerName,
		Labels:         map[string]string{"k8s/controller-name": d.opts.ControllerName},
		BaseURL:        url,
		AdminKey:       d.opts.AdminKey,
	})
	if err != nil {
		return fmt.Errorf("adding cluster: %w", err)
	}

	httpsURL := fmt.Sprintf("https://%s/apisix/admin", d.framework.GetDashboardEndpointHTTPS())
	err = d.apisixCli.AddCluster(context.Background(), &dashboard.ClusterOptions{
		Name:          "default-https",
		BaseURL:       httpsURL,
		AdminKey:      d.opts.AdminKey,
		SkipTLSVerify: true,
	})
	if err != nil {
		return fmt.Errorf("adding https cluster: %w", err)
	}

	return nil
}

// createTunnels creates HTTP and HTTPS tunnels for API7 gateway
func (d *API7Deployer) createTunnels() {
	// TODO: Implement tunnel creation logic
	// This should create tunnels to the API7 gateway service
}

// newHTTPSClientWithCertificates creates HTTPS client with certificates
func (d *API7Deployer) newHTTPSClientWithCertificates(
	host string, insecure bool, ca *x509.CertPool, certs []tls.Certificate,
) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   d.httpsTunnel.Endpoint(),
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
