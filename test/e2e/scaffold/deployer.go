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
	"fmt"
	"os"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// DeployMode defines the deployment mode
type DeployMode string

const (
	DeployModeAPI7   DeployMode = "api7"
	DeployModeAPISIX DeployMode = "apisix"
)

// Deployer defines the interface for deploying data plane components
type Deployer interface {
	// Deploy deploys the data plane components
	// for api7 mode: deploy api7 dashboard and api7 gateway
	// for apisix mode: deploy apisix dp only
	Deploy(ctx context.Context) error

	// Cleanup cleans up deployed resources
	Cleanup(ctx context.Context) error

	// GetHTTPClient returns HTTP client for the gateway data plane
	GetHTTPClient() *httpexpect.Expect

	// GetHTTPSClient returns HTTPS client for the gateway data plane
	GetHTTPSClient(host string) *httpexpect.Expect

	// GetAdminClient returns admin client for configuration
	GetAdminClient() *httpexpect.Expect

	// GetAdminKey returns admin key for authentication
	GetAdminKey() string

	// GetDataplaneResource returns dashboard cluster for admin operations
	GetDataplaneResource() dashboard.Cluster
	GetDataplaneResourceHTTPS() dashboard.Cluster

	// GetMode returns the deployment mode
	GetMode() DeployMode
}

// DeployerFactory creates deployers based on mode
type DeployerFactory interface {
	CreateDeployer(mode DeployMode, opts *DeployerOptions) (Deployer, error)
}

// DeployerOptions contains options for deployer creation
type DeployerOptions struct {
	Namespace      string
	AdminKey       string
	ControllerName string
	// API7-specific options
	GatewayGroupID string
	DashboardAddr  string
	// APISIX-specific options
	APISIXImage string
	ServiceType string
}

// defaultDeployerFactory is the default factory implementation
type defaultDeployerFactory struct {
	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	framework   interface{}
}

// NewDeployerFactory creates a new deployer factory
func NewDeployerFactory(t testing.TestingT, kubectlOpts *k8s.KubectlOptions, framework interface{}) DeployerFactory {
	return &defaultDeployerFactory{
		t:           t,
		kubectlOpts: kubectlOpts,
		framework:   framework,
	}
}

// CreateDeployer creates a deployer based on the specified mode
func (f *defaultDeployerFactory) CreateDeployer(mode DeployMode, opts *DeployerOptions) (Deployer, error) {
	switch mode {
	case DeployModeAPI7:
		// Extract API7 framework
		api7Framework, ok := f.framework.(*framework.Framework)
		if !ok {
			return nil, fmt.Errorf("invalid framework type for API7 mode")
		}
		return NewAPI7Deployer(f.t, f.kubectlOpts, api7Framework, opts)
	case DeployModeAPISIX:
		// Extract APISIX framework
		apisixFramework, ok := f.framework.(*framework.APISIXFramework)
		if !ok {
			return nil, fmt.Errorf("invalid framework type for APISIX mode")
		}
		return NewAPISIXDeployer(f.t, f.kubectlOpts, apisixFramework, opts)
	default:
		return nil, fmt.Errorf("unsupported deploy mode: %s", mode)
	}
}

// GetDeployModeFromEnv returns deployment mode from environment variable
func GetDeployModeFromEnv() DeployMode {
	mode := os.Getenv("DEPLOY_MODE")
	switch mode {
	case "api7":
		return DeployModeAPI7
	case "apisix":
		return DeployModeAPISIX
	default:
		// Default to API7 for backward compatibility
		return DeployModeAPI7
	}
}
