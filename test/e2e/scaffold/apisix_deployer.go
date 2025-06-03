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
	"net/http"
	"net/url"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// APISIXDeployer implements Deployer interface for APISIX standalone
type APISIXDeployer struct {
	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	framework   *framework.APISIXFramework
	opts        *DeployerOptions

	// APISIX-specific resources
	service     *corev1.Service
	httpTunnel  *k8s.Tunnel
	httpsTunnel *k8s.Tunnel
	adminClient *httpexpect.Expect
}

// NewAPISIXDeployer creates a new APISIX deployer
func NewAPISIXDeployer(t testing.TestingT, kubectlOpts *k8s.KubectlOptions, framework *framework.APISIXFramework, opts *DeployerOptions) (Deployer, error) {
	if opts == nil {
		return nil, fmt.Errorf("deployer options cannot be nil")
	}

	return &APISIXDeployer{
		t:           t,
		kubectlOpts: kubectlOpts,
		framework:   framework,
		opts:        opts,
	}, nil
}

// Deploy deploys APISIX in standalone mode
func (d *APISIXDeployer) Deploy(ctx context.Context) error {
	// Deploy APISIX standalone using framework
	deployer := framework.NewAPISIXDeployer(d.t, d.kubectlOpts, &framework.APISIXDeployOptions{
		Namespace:   d.opts.Namespace,
		Image:       d.opts.APISIXImage,
		ServiceType: d.opts.ServiceType,
		AdminKey:    d.opts.AdminKey,
	})

	err := deployer.Deploy(ctx)
	if err != nil {
		return fmt.Errorf("deploying APISIX: %w", err)
	}

	// Get the service
	d.service = deployer.GetService()

	// Create tunnels
	d.createTunnels()

	// Initialize admin client
	d.initAdminClient()

	return nil
}

// Cleanup cleans up APISIX resources
func (d *APISIXDeployer) Cleanup(ctx context.Context) error {
	// Close tunnels
	if d.httpTunnel != nil {
		d.httpTunnel.Close()
	}
	if d.httpsTunnel != nil {
		d.httpsTunnel.Close()
	}

	// Delete APISIX deployment and service
	// This will be handled by namespace deletion
	return nil
}

// GetHTTPClient returns HTTP client for APISIX gateway
func (d *APISIXDeployer) GetHTTPClient() *httpexpect.Expect {
	if d.httpTunnel == nil {
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

// GetHTTPSClient returns HTTPS client for APISIX gateway
func (d *APISIXDeployer) GetHTTPSClient(host string) *httpexpect.Expect {
	if d.httpsTunnel == nil {
		d.createTunnels()
	}

	u := url.URL{
		Scheme: "https",
		Host:   d.httpsTunnel.Endpoint(),
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

// GetAdminClient returns admin client for APISIX
func (d *APISIXDeployer) GetAdminClient() *httpexpect.Expect {
	if d.adminClient == nil {
		d.initAdminClient()
	}
	return d.adminClient
}

// GetAdminKey returns admin key for APISIX
func (d *APISIXDeployer) GetAdminKey() string {
	return d.opts.AdminKey
}

// GetDataplaneResource returns dashboard cluster for admin operations
// For APISIX standalone, this creates a virtual cluster that directly talks to APISIX admin API
func (d *APISIXDeployer) GetDataplaneResource() dashboard.Cluster {
	// TODO: Implement virtual cluster for APISIX standalone
	// This should create a dashboard.Cluster that talks directly to APISIX admin API
	return nil
}

// GetDataplaneResourceHTTPS returns HTTPS dashboard cluster
func (d *APISIXDeployer) GetDataplaneResourceHTTPS() dashboard.Cluster {
	// TODO: Implement virtual HTTPS cluster for APISIX standalone
	return nil
}

// GetMode returns the deployment mode
func (d *APISIXDeployer) GetMode() DeployMode {
	return DeployModeAPISIX
}

// createTunnels creates HTTP and HTTPS tunnels for APISIX
func (d *APISIXDeployer) createTunnels() {
	if d.service == nil {
		return
	}

	// Create HTTP tunnel
	httpTunnel := k8s.NewTunnel(d.kubectlOpts, k8s.ResourceTypeService, d.service.Name, 9080, 9080)
	httpTunnel.ForwardPort(d.t)
	d.httpTunnel = httpTunnel

	// Create HTTPS tunnel
	httpsTunnel := k8s.NewTunnel(d.kubectlOpts, k8s.ResourceTypeService, d.service.Name, 9443, 9443)
	httpsTunnel.ForwardPort(d.t)
	d.httpsTunnel = httpsTunnel
}

// initAdminClient initializes the admin client
func (d *APISIXDeployer) initAdminClient() {
	if d.service == nil {
		return
	}

	// Create admin tunnel
	adminTunnel := k8s.NewTunnel(d.kubectlOpts, k8s.ResourceTypeService, d.service.Name, 9180, 9180)
	adminTunnel.ForwardPort(d.t)

	u := url.URL{
		Scheme: "http",
		Host:   adminTunnel.Endpoint(),
	}
	d.adminClient = httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}
