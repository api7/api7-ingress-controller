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
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:embed manifests/apisix-standalone.yaml
var apisixStandaloneTemplate string

// APISIXDeployOptions contains options for APISIX standalone deployment
type APISIXDeployOptions struct {
	Namespace     string
	Image         string
	AdminKey      string
	ServiceType   string
	HTTPNodePort  int32
	HTTPSNodePort int32
	AdminNodePort int32
}

// APISIXDeployer implements DataPlaneDeployer for APISIX standalone
type APISIXDeployer struct {
	kubectlOpts *k8s.KubectlOptions
	opts        *APISIXDeployOptions
	service     *corev1.Service
	t           testing.TestingT
}

// NewAPISIXDeployer creates a new APISIX deployer
func NewAPISIXDeployer(t testing.TestingT, kubectlOpts *k8s.KubectlOptions, opts *APISIXDeployOptions) *APISIXDeployer {
	if opts.Image == "" {
		opts.Image = getEnvOrDefault("APISIX_IMAGE", "apache/apisix:dev")
	}
	if opts.AdminKey == "" {
		opts.AdminKey = getEnvOrDefault("APISIX_ADMIN_KEY", "edd1c9f034335f136f87ad84b625c8f1")
	}
	if opts.Namespace == "" {
		opts.Namespace = getEnvOrDefault("APISIX_NAMESPACE", "apisix-standalone")
	}
	if opts.ServiceType == "" {
		opts.ServiceType = "ClusterIP"
	}

	return &APISIXDeployer{
		kubectlOpts: kubectlOpts,
		opts:        opts,
		t:           t,
	}
}

func (d *APISIXDeployer) GetService() *corev1.Service {
	return d.service
}

// Deploy deploys APISIX standalone
func (d *APISIXDeployer) Deploy(ctx context.Context) error {
	// Parse and execute template
	tmpl, err := template.New("apisix-standalone").Funcs(sprig.TxtFuncMap()).Parse(apisixStandaloneTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d.opts); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Apply the manifest
	if err := k8s.KubectlApplyFromStringE(d.t, d.kubectlOpts, buf.String()); err != nil {
		return fmt.Errorf("failed to apply APISIX manifest: %w", err)
	}

	// Wait for deployment to be ready
	if err := d.waitForDeployment(ctx); err != nil {
		return fmt.Errorf("failed to wait for deployment: %w", err)
	}

	// Get service
	service, err := k8s.GetServiceE(d.t, d.kubectlOpts, "apisix")
	if err != nil {
		return fmt.Errorf("failed to get APISIX service: %w", err)
	}
	d.service = service

	return nil
}

// Cleanup removes APISIX standalone deployment
func (d *APISIXDeployer) Cleanup(ctx context.Context) error {
	// Delete namespace which will clean up all resources
	return k8s.DeleteNamespaceE(d.t, d.kubectlOpts, d.opts.Namespace)
}

// waitForDeployment waits for the APISIX deployment to be ready
func (d *APISIXDeployer) waitForDeployment(ctx context.Context) error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		pods, err := k8s.ListPodsE(d.t, d.kubectlOpts, metav1.ListOptions{
			LabelSelector: "app=apisix",
		})
		if err != nil {
			return false, err
		}

		if len(pods) == 0 {
			return false, nil
		}

		for _, pod := range pods {
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			// Check if all containers are ready
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
		}

		return true, nil
	})
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
