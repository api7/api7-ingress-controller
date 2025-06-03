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
	"encoding/base64"
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeTlsSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: kubernetes.io/tls
data:
  tls.crt: %s
  tls.key: %s
`
)

// BaseScaffold provides common implementation for scaffold methods
type BaseScaffold struct {
	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	namespace   string
	opts        *Options
}

// Common resource management methods that can be shared between scaffolds

// CreateResourceFromString creates a K8s resource from YAML string
func (b *BaseScaffold) CreateResourceFromString(resourceYaml string) error {
	return b.CreateResourceFromStringWithNamespace(resourceYaml, b.namespace)
}

// CreateResourceFromStringWithNamespace creates a K8s resource from YAML string in specified namespace
func (b *BaseScaffold) CreateResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	kubectlOpts := *b.kubectlOpts
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	return k8s.KubectlApplyFromStringE(b.t, &kubectlOpts, resourceYaml)
}

// DeleteResourceFromString deletes a K8s resource from YAML string
func (b *BaseScaffold) DeleteResourceFromString(resourceYaml string) error {
	return b.DeleteResourceFromStringWithNamespace(resourceYaml, b.namespace)
}

// DeleteResourceFromStringWithNamespace deletes a K8s resource from YAML string in specified namespace
func (b *BaseScaffold) DeleteResourceFromStringWithNamespace(resourceYaml, namespace string) error {
	kubectlOpts := *b.kubectlOpts
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	return k8s.KubectlDeleteFromStringE(b.t, &kubectlOpts, resourceYaml)
}

// DeleteResource deletes a K8s resource by type and name
func (b *BaseScaffold) DeleteResource(resourceType, name string) error {
	args := []string{"delete", resourceType, name}
	return k8s.RunKubectlE(b.t, b.kubectlOpts, args...)
}

// GetResourceYaml gets a K8s resource YAML by type and name
func (b *BaseScaffold) GetResourceYaml(resourceType, name string) (string, error) {
	return b.GetResourceYamlFromNamespace(resourceType, name, b.namespace)
}

// GetResourceYamlFromNamespace gets a K8s resource YAML by type, name and namespace
func (b *BaseScaffold) GetResourceYamlFromNamespace(resourceType, name, namespace string) (string, error) {
	kubectlOpts := *b.kubectlOpts
	if namespace != "" {
		kubectlOpts.Namespace = namespace
	}
	args := []string{"get", resourceType, name, "-o", "yaml"}
	return k8s.RunKubectlAndGetOutputE(b.t, &kubectlOpts, args...)
}

// RunKubectlAndGetOutput runs kubectl command and returns output
func (b *BaseScaffold) RunKubectlAndGetOutput(args ...string) (string, error) {
	return k8s.RunKubectlAndGetOutputE(b.t, b.kubectlOpts, args...)
}

// NewKubeTlsSecret creates a TLS secret
func (b *BaseScaffold) NewKubeTlsSecret(secretName, cert, key string) error {
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))
	secret := fmt.Sprintf(kubeTlsSecretTemplate, secretName, certBase64, keyBase64)
	return b.CreateResourceFromString(secret)
}

// Namespace returns the current namespace
func (b *BaseScaffold) Namespace() string {
	return b.namespace
}

// GetContext returns the context
func (b *BaseScaffold) GetContext() context.Context {
	return context.TODO()
}

// GetGinkgoT returns the Ginkgo test interface
func (b *BaseScaffold) GetGinkgoT() ginkgo.GinkgoTInterface {
	return ginkgo.GinkgoT()
}

// GetK8sClient returns the Kubernetes client
func (b *BaseScaffold) GetK8sClient() client.Client {
	// This needs to be implemented by the concrete scaffold implementations
	// since they have access to the framework with the client
	panic("GetK8sClient not implemented in BaseScaffold - should be implemented by concrete scaffolds")
}

// GetControllerName returns the controller name
func (b *BaseScaffold) GetControllerName() string {
	if b.opts != nil && b.opts.ControllerName != "" {
		return b.opts.ControllerName
	}
	return DefaultControllerName
}

// AdminKey returns the admin key
func (b *BaseScaffold) AdminKey() string {
	if b.opts != nil && b.opts.APISIXAdminAPIKey != "" {
		return b.opts.APISIXAdminAPIKey
	}
	return "edd1c9f034335f136f87ad84b625c8f1" // Default APISIX admin key
}

// TODO: These methods need specific implementations in each scaffold
// and should not be in the base scaffold as they depend on the deployment mode

// ResourceApplied is a placeholder - should be implemented in specific scaffolds
func (b *BaseScaffold) ResourceApplied(resourType, resourceName, resourceRaw string, observedGeneration int) {
	// This method needs specific implementation based on deployment mode
}

// ApplyDefaultGatewayResource is a placeholder - should be implemented in specific scaffolds
func (b *BaseScaffold) ApplyDefaultGatewayResource(gatewayProxy, gatewayClass, gateway, httpRoute string) {
	// This method needs specific implementation based on deployment mode
}

// ScaleIngress is a placeholder - should be implemented in specific scaffolds
func (b *BaseScaffold) ScaleIngress(replicas int) {
	// This method needs specific implementation based on deployment mode
}
