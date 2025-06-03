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

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// VirtualCluster implements dashboard.Cluster interface for APISIX standalone mode
// It provides a virtual cluster that talks directly to APISIX admin API
// For now, it's a minimal implementation that returns "not supported" errors
type VirtualCluster struct {
	name        string
	adminClient *httpexpect.Expect
	adminKey    string
	baseURL     string
	httpsMode   bool
}

// VirtualCluster methods that implement dashboard.Cluster interface
func (v *VirtualCluster) Route() dashboard.Route {
	return &unsupportedRoute{}
}

func (v *VirtualCluster) Service() dashboard.Service {
	return &unsupportedService{}
}

func (v *VirtualCluster) SSL() dashboard.SSL {
	return &unsupportedSSL{}
}

func (v *VirtualCluster) StreamRoute() dashboard.StreamRoute {
	return &unsupportedStreamRoute{}
}

func (v *VirtualCluster) GlobalRule() dashboard.GlobalRule {
	return &unsupportedGlobalRule{}
}

func (v *VirtualCluster) Consumer() dashboard.Consumer {
	return &unsupportedConsumer{}
}

func (v *VirtualCluster) Plugin() dashboard.Plugin {
	return &unsupportedPlugin{}
}

func (v *VirtualCluster) PluginConfig() dashboard.PluginConfig {
	return &unsupportedPluginConfig{}
}

func (v *VirtualCluster) Schema() dashboard.Schema {
	return &unsupportedSchema{}
}

func (v *VirtualCluster) PluginMetadata() dashboard.PluginMetadata {
	return &unsupportedPluginMetadata{}
}

func (v *VirtualCluster) Validator() dashboard.APISIXSchemaValidator {
	return &unsupportedValidator{}
}

func (v *VirtualCluster) String() string {
	return fmt.Sprintf("virtual cluster %s", v.name)
}

func (v *VirtualCluster) HasSynced(ctx context.Context) error {
	// For APISIX standalone, we always consider it synced
	return nil
}

func (v *VirtualCluster) HealthCheck(ctx context.Context) error {
	// Simple health check via admin API
	resp := v.adminClient.GET("/apisix/admin").
		WithHeader("X-API-KEY", v.adminKey).
		Expect()
	if resp.Raw().StatusCode >= 200 && resp.Raw().StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("APISIX admin API health check failed with status %d", resp.Raw().StatusCode)
}

// Unsupported implementations that return "not supported" errors
// These can be implemented later if specific tests require them

type unsupportedRoute struct{}

func (u *unsupportedRoute) Get(ctx context.Context, name string) (*v1.Route, error) {
	return nil, fmt.Errorf("route operations not supported in virtual cluster")
}
func (u *unsupportedRoute) List(ctx context.Context, args ...any) ([]*v1.Route, error) {
	return nil, fmt.Errorf("route operations not supported in virtual cluster")
}
func (u *unsupportedRoute) Create(ctx context.Context, route *v1.Route) (*v1.Route, error) {
	return nil, fmt.Errorf("route operations not supported in virtual cluster")
}
func (u *unsupportedRoute) Delete(ctx context.Context, route *v1.Route) error {
	return fmt.Errorf("route operations not supported in virtual cluster")
}
func (u *unsupportedRoute) Update(ctx context.Context, route *v1.Route) (*v1.Route, error) {
	return nil, fmt.Errorf("route operations not supported in virtual cluster")
}

type unsupportedService struct{}

func (u *unsupportedService) Get(ctx context.Context, name string) (*v1.Service, error) {
	return nil, fmt.Errorf("service operations not supported in virtual cluster")
}
func (u *unsupportedService) List(ctx context.Context, args ...any) ([]*v1.Service, error) {
	return nil, fmt.Errorf("service operations not supported in virtual cluster")
}
func (u *unsupportedService) Create(ctx context.Context, svc *v1.Service) (*v1.Service, error) {
	return nil, fmt.Errorf("service operations not supported in virtual cluster")
}
func (u *unsupportedService) Delete(ctx context.Context, svc *v1.Service) error {
	return fmt.Errorf("service operations not supported in virtual cluster")
}
func (u *unsupportedService) Update(ctx context.Context, svc *v1.Service) (*v1.Service, error) {
	return nil, fmt.Errorf("service operations not supported in virtual cluster")
}

type unsupportedSSL struct{}

func (u *unsupportedSSL) Get(ctx context.Context, name string) (*v1.Ssl, error) {
	return nil, fmt.Errorf("SSL operations not supported in virtual cluster")
}
func (u *unsupportedSSL) List(ctx context.Context, args ...any) ([]*v1.Ssl, error) {
	return nil, fmt.Errorf("SSL operations not supported in virtual cluster")
}
func (u *unsupportedSSL) Create(ctx context.Context, ssl *v1.Ssl) (*v1.Ssl, error) {
	return nil, fmt.Errorf("SSL operations not supported in virtual cluster")
}
func (u *unsupportedSSL) Delete(ctx context.Context, ssl *v1.Ssl) error {
	return fmt.Errorf("SSL operations not supported in virtual cluster")
}
func (u *unsupportedSSL) Update(ctx context.Context, ssl *v1.Ssl) (*v1.Ssl, error) {
	return nil, fmt.Errorf("SSL operations not supported in virtual cluster")
}

type unsupportedStreamRoute struct{}

func (u *unsupportedStreamRoute) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	return nil, fmt.Errorf("stream route operations not supported in virtual cluster")
}
func (u *unsupportedStreamRoute) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	return nil, fmt.Errorf("stream route operations not supported in virtual cluster")
}
func (u *unsupportedStreamRoute) Create(ctx context.Context, route *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, fmt.Errorf("stream route operations not supported in virtual cluster")
}
func (u *unsupportedStreamRoute) Delete(ctx context.Context, route *v1.StreamRoute) error {
	return fmt.Errorf("stream route operations not supported in virtual cluster")
}
func (u *unsupportedStreamRoute) Update(ctx context.Context, route *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, fmt.Errorf("stream route operations not supported in virtual cluster")
}

type unsupportedGlobalRule struct{}

func (u *unsupportedGlobalRule) Get(ctx context.Context, id string) (*v1.GlobalRule, error) {
	return nil, fmt.Errorf("global rule operations not supported in virtual cluster")
}
func (u *unsupportedGlobalRule) List(ctx context.Context) ([]*v1.GlobalRule, error) {
	return nil, fmt.Errorf("global rule operations not supported in virtual cluster")
}
func (u *unsupportedGlobalRule) Create(ctx context.Context, rule *v1.GlobalRule) (*v1.GlobalRule, error) {
	return nil, fmt.Errorf("global rule operations not supported in virtual cluster")
}
func (u *unsupportedGlobalRule) Delete(ctx context.Context, rule *v1.GlobalRule) error {
	return fmt.Errorf("global rule operations not supported in virtual cluster")
}
func (u *unsupportedGlobalRule) Update(ctx context.Context, rule *v1.GlobalRule) (*v1.GlobalRule, error) {
	return nil, fmt.Errorf("global rule operations not supported in virtual cluster")
}

type unsupportedConsumer struct{}

func (u *unsupportedConsumer) Get(ctx context.Context, name string) (*v1.Consumer, error) {
	return nil, fmt.Errorf("consumer operations not supported in virtual cluster")
}
func (u *unsupportedConsumer) List(ctx context.Context) ([]*v1.Consumer, error) {
	return nil, fmt.Errorf("consumer operations not supported in virtual cluster")
}
func (u *unsupportedConsumer) Create(ctx context.Context, consumer *v1.Consumer) (*v1.Consumer, error) {
	return nil, fmt.Errorf("consumer operations not supported in virtual cluster")
}
func (u *unsupportedConsumer) Delete(ctx context.Context, consumer *v1.Consumer) error {
	return fmt.Errorf("consumer operations not supported in virtual cluster")
}
func (u *unsupportedConsumer) Update(ctx context.Context, consumer *v1.Consumer) (*v1.Consumer, error) {
	return nil, fmt.Errorf("consumer operations not supported in virtual cluster")
}

type unsupportedPlugin struct{}

func (u *unsupportedPlugin) List(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("plugin operations not supported in virtual cluster")
}

type unsupportedPluginConfig struct{}

func (u *unsupportedPluginConfig) Get(ctx context.Context, name string) (*v1.PluginConfig, error) {
	return nil, fmt.Errorf("plugin config operations not supported in virtual cluster")
}
func (u *unsupportedPluginConfig) List(ctx context.Context) ([]*v1.PluginConfig, error) {
	return nil, fmt.Errorf("plugin config operations not supported in virtual cluster")
}
func (u *unsupportedPluginConfig) Create(ctx context.Context, plugin *v1.PluginConfig) (*v1.PluginConfig, error) {
	return nil, fmt.Errorf("plugin config operations not supported in virtual cluster")
}
func (u *unsupportedPluginConfig) Delete(ctx context.Context, plugin *v1.PluginConfig) error {
	return fmt.Errorf("plugin config operations not supported in virtual cluster")
}
func (u *unsupportedPluginConfig) Update(ctx context.Context, plugin *v1.PluginConfig) (*v1.PluginConfig, error) {
	return nil, fmt.Errorf("plugin config operations not supported in virtual cluster")
}

type unsupportedSchema struct{}

func (u *unsupportedSchema) GetPluginSchema(ctx context.Context, pluginName string) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}
func (u *unsupportedSchema) GetRouteSchema(ctx context.Context) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}
func (u *unsupportedSchema) GetUpstreamSchema(ctx context.Context) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}
func (u *unsupportedSchema) GetConsumerSchema(ctx context.Context) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}
func (u *unsupportedSchema) GetSslSchema(ctx context.Context) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}
func (u *unsupportedSchema) GetPluginConfigSchema(ctx context.Context) (*v1.Schema, error) {
	return nil, fmt.Errorf("schema operations not supported in virtual cluster")
}

type unsupportedPluginMetadata struct{}

func (u *unsupportedPluginMetadata) Get(ctx context.Context, name string) (*v1.PluginMetadata, error) {
	return nil, fmt.Errorf("plugin metadata operations not supported in virtual cluster")
}
func (u *unsupportedPluginMetadata) List(ctx context.Context) ([]*v1.PluginMetadata, error) {
	return nil, fmt.Errorf("plugin metadata operations not supported in virtual cluster")
}
func (u *unsupportedPluginMetadata) Create(ctx context.Context, metadata *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	return nil, fmt.Errorf("plugin metadata operations not supported in virtual cluster")
}
func (u *unsupportedPluginMetadata) Delete(ctx context.Context, metadata *v1.PluginMetadata) error {
	return fmt.Errorf("plugin metadata operations not supported in virtual cluster")
}
func (u *unsupportedPluginMetadata) Update(ctx context.Context, metadata *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	return nil, fmt.Errorf("plugin metadata operations not supported in virtual cluster")
}

type unsupportedValidator struct{}

func (u *unsupportedValidator) ValidateStreamPluginSchema(plugins v1.Plugins) (bool, error) {
	return false, fmt.Errorf("validation not supported in virtual cluster")
}
func (u *unsupportedValidator) ValidateHTTPPluginSchema(plugins v1.Plugins) (bool, error) {
	return false, fmt.Errorf("validation not supported in virtual cluster")
}

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
	frameworkDeployer := framework.NewAPISIXDeployer(d.t, d.kubectlOpts, &framework.APISIXDeployOptions{
		Namespace:   d.opts.Namespace,
		Image:       d.opts.APISIXImage,
		ServiceType: d.opts.ServiceType,
		AdminKey:    d.opts.AdminKey,
	})

	err := frameworkDeployer.Deploy(ctx)
	if err != nil {
		return fmt.Errorf("deploying APISIX: %w", err)
	}

	// Get the service
	d.service = frameworkDeployer.GetService()

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
	return &VirtualCluster{
		name:        "default",
		adminClient: d.GetAdminClient(),
		adminKey:    d.GetAdminKey(),
		baseURL:     fmt.Sprintf("http://%s", d.getAdminEndpoint()),
	}
}

// GetDataplaneResourceHTTPS returns HTTPS dashboard cluster
func (d *APISIXDeployer) GetDataplaneResourceHTTPS() dashboard.Cluster {
	return &VirtualCluster{
		name:        "default-https",
		adminClient: d.GetAdminClient(), // For now, use the same admin client
		adminKey:    d.GetAdminKey(),
		baseURL:     fmt.Sprintf("https://%s", d.getAdminEndpoint()),
		httpsMode:   true,
	}
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

// getAdminEndpoint returns the admin endpoint for APISIX
func (d *APISIXDeployer) getAdminEndpoint() string {
	// Create admin tunnel if not exists
	if d.service != nil {
		adminTunnel := k8s.NewTunnel(d.kubectlOpts, k8s.ResourceTypeService, d.service.Name, 0, 9180)
		adminTunnel.ForwardPort(d.t)
		return adminTunnel.Endpoint()
	}
	return "localhost:9180"
}
