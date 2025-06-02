package scaffold

import (
	"context"

	"github.com/gavv/httpexpect/v2"
	"github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/apisix-ingress-controller/pkg/dashboard"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

var NewScaffold func(*Options) TestScaffold

// TestScaffold defines the interface for test scaffold implementations
type TestScaffold interface {
	// HTTP client methods
	NewAPISIXClient() *httpexpect.Expect
	NewAPISIXHttpsClient(host string) *httpexpect.Expect

	// Basic operation methods
	AdminKey() string
	GetControllerName() string
	Namespace() string
	GetContext() context.Context
	GetGinkgoT() ginkgo.GinkgoTInterface
	GetK8sClient() client.Client

	// Resource management methods
	CreateResourceFromString(resourceYaml string) error
	CreateResourceFromStringWithNamespace(resourceYaml, namespace string) error
	DeleteResourceFromString(resourceYaml string) error
	DeleteResourceFromStringWithNamespace(resourceYaml, namespace string) error
	DeleteResource(resourceType, name string) error
	GetResourceYaml(resourceType, name string) (string, error)
	GetResourceYamlFromNamespace(resourceType, name, namespace string) (string, error)
	ResourceApplied(resourType, resourceName, resourceRaw string, observedGeneration int)
	ApplyDefaultGatewayResource(gatewayProxy, gatewayClass, gateway, httpRoute string)
	GetDeploymentLogs(name string) string

	// Kubernetes operation methods
	RunKubectlAndGetOutput(args ...string) (string, error)
	NewKubeTlsSecret(secretName, cert, key string) error

	// Dataplane resource access methods
	DefaultDataplaneResource() dashboard.Cluster
	DefaultDataplaneResourceHTTPS() dashboard.Cluster

	// TODO: remove it
	// Gateway group management methods (for multi-gateway support)
	CreateAdditionalGatewayGroup(namePrefix string) (string, string, error)
	GetAdditionalGatewayGroup(gatewayGroupID string) (*GatewayGroupResources, bool)
	NewAPISIXClientForGatewayGroup(gatewayGroupID string) (*httpexpect.Expect, error)

	ScaleIngress(int)
	DeployNginx(options framework.NginxOptions)
}

//// Deployer defines the interface for deploying data plane components
//type Deployer interface {
//	// for api7 mode, deploy api7 dashboard and api7 gateway
//	// fror apisix mode, deploy apisix dp only
//	Deploy(ctx context.Context, opts *DeployOptions) (*DeployResult, error)
//
//	// GetClient returns HTTP client for the gateway
//	GetHTTPClient() *httpexpect.Expect
//
//	// GetHTTPSClient returns HTTPS client for the gateway
//	GetHTTPSClient() *httpexpect.Expect
//
//	GetAdminHTTPClient() ...
//
//	GetAdminKey() ...
//}

//type TestFrameWork interface {
//	NewScaffold(*Options) TestScaffold
//}
