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
	// HTTP client methods - common interface but implementation may differ
	NewAPISIXClient() *httpexpect.Expect
	NewAPISIXHttpsClient(host string) *httpexpect.Expect

	// Basic operation methods - common to all implementations
	AdminKey() string
	GetControllerName() string
	Namespace() string
	GetContext() context.Context
	GetGinkgoT() ginkgo.GinkgoTInterface
	GetK8sClient() client.Client

	// Resource management methods - common to all implementations
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

	// Kubernetes operation methods - common to all implementations
	RunKubectlAndGetOutput(args ...string) (string, error)
	NewKubeTlsSecret(secretName, cert, key string) error

	// Dataplane resource access methods - common interface
	DefaultDataplaneResource() dashboard.Cluster
	DefaultDataplaneResourceHTTPS() dashboard.Cluster

	// Common infrastructure methods
	ScaleIngress(int)
	DeployNginx(options framework.NginxOptions)

	// Access to underlying deployer
	GetDeployer() Deployer

	// TODO: These methods should be deprecated for multi-gateway support
	// They are API7-specific and should be moved to a separate interface
	CreateAdditionalGatewayGroup(namePrefix string) (string, string, error)
	GetAdditionalGatewayGroup(gatewayGroupID string) (*GatewayGroupResources, bool)
	NewAPISIXClientForGatewayGroup(gatewayGroupID string) (*httpexpect.Expect, error)
}

//type TestFrameWork interface {
//	NewScaffold(*Options) TestScaffold
//}
