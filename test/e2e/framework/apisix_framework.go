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

package framework

import (
	"context"
	"os"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_apisixNamespace = "apisix-standalone-e2e"
	_apisixFramework *APISIXFramework
)

// APISIXFramework implements TestFramework for APISIX standalone
type APISIXFramework struct {
	Context context.Context
	GinkgoT GinkgoTInterface
	GomegaT *GomegaWithT

	Logger logger.TestLogger

	kubectlOpts *k8s.KubectlOptions
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config
	K8sClient   client.Client
}

// NewAPISIXFramework creates a new APISIX framework
func NewAPISIXFramework() *APISIXFramework {
	f := &APISIXFramework{
		GinkgoT: GinkgoT(),
		GomegaT: NewWithT(GinkgoT(4)),
		Logger:  logger.Terratest,
	}

	f.Context = context.TODO()

	// Use environment variable for namespace if set
	namespace := os.Getenv("APISIX_NAMESPACE")
	if namespace == "" {
		namespace = _apisixNamespace
	}

	f.kubectlOpts = k8s.NewKubectlOptions("", "", namespace)
	restCfg, err := buildRestConfig("")
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "building API Server rest config")
	f.restConfig = restCfg

	clientset, err := kubernetes.NewForConfig(restCfg)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating Kubernetes clientset")
	f.clientset = clientset

	k8sClient, err := client.New(restCfg, client.Options{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating controller-runtime client")
	f.K8sClient = k8sClient

	_apisixFramework = f

	return f
}

// BeforeSuite initializes the APISIX test environment
func (f *APISIXFramework) BeforeSuite() {
	f.Logf("Starting APISIX standalone test suite")

	// Create namespace for APISIX standalone tests
	k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, f.kubectlOpts.Namespace)

	f.Logf("APISIX standalone test environment initialized")
}

// AfterSuite cleans up the APISIX test environment
func (f *APISIXFramework) AfterSuite() {
	f.Logf("Cleaning up APISIX standalone test environment")

	// Clean up namespace
	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, f.kubectlOpts.Namespace)
}

// GetFramework returns the global APISIX framework instance
func GetAPISIXFramework() *APISIXFramework {
	return _apisixFramework
}

// Logf logs a formatted message
func (f *APISIXFramework) Logf(format string, v ...any) {
	f.Logger.Logf(f.GinkgoT, format, v...)
}
