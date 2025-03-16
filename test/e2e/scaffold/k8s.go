// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/api7/api7-ingress-controller/pkg/dashboard"
	apisix "github.com/api7/api7-ingress-controller/pkg/dashboard"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateResourceFromString creates resource from a loaded yaml string.
func (s *Scaffold) CreateResourceFromString(yaml string) error {
	err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, yaml)
	// if the error raised, it may be a &shell.ErrWithCmdOutput, which is useless in debug
	if err != nil {
		err = fmt.Errorf(err.Error())
	}
	return err
}

func (s *Scaffold) DeleteResourceFromString(yaml string) error {
	return k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
}

func (s *Scaffold) Exec(podName, containerName string, args ...string) (string, error) {
	cmdArgs := []string{}

	if s.kubectlOptions.ContextName != "" {
		cmdArgs = append(cmdArgs, "--context", s.kubectlOptions.ContextName)
	}
	if s.kubectlOptions.ConfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", s.kubectlOptions.ConfigPath)
	}
	if s.kubectlOptions.Namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", s.kubectlOptions.Namespace)
	}

	cmdArgs = append(cmdArgs, "exec")
	cmdArgs = append(cmdArgs, "-i")
	cmdArgs = append(cmdArgs, podName)
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, containerName)
	cmdArgs = append(cmdArgs, "--", "sh", "-c")
	cmdArgs = append(cmdArgs, args...)

	GinkgoWriter.Printf("running command: kubectl %v\n", strings.Join(cmdArgs, " "))

	output, err := exec.Command("kubectl", cmdArgs...).Output()

	return strings.TrimSuffix(string(output), "\n"), err
}

func (s *Scaffold) GetOutputFromString(shell ...string) (string, error) {
	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, shell...)
	output, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, cmdArgs...)
	return output, err
}

func (s *Scaffold) GetResourceYamlFromNamespace(resourceType, resourceName, namespace string) (string, error) {
	return s.GetOutputFromString(resourceType, resourceName, "-n", namespace, "-o", "yaml")
}

func (s *Scaffold) GetResourceYaml(resourceType, resourceName string) (string, error) {
	return s.GetOutputFromString(resourceType, resourceName, "-o", "yaml")
}

// RemoveResourceByString remove resource from a loaded yaml string.
func (s *Scaffold) RemoveResourceByString(yaml string) error {
	err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
	time.Sleep(5 * time.Second)
	return err
}

func (s *Scaffold) GetServiceByName(name string) (*corev1.Service, error) {
	return k8s.GetServiceE(s.t, s.kubectlOptions, name)
}

// ListPodsByLabels lists all pods which matching the label selector.
func (s *Scaffold) ListPodsByLabels(labels string) ([]corev1.Pod, error) {
	return k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: labels,
	})
}

// CreateResourceFromStringWithNamespace creates resource from a loaded yaml string
// and sets its namespace to the specified one.
func (s *Scaffold) CreateResourceFromStringWithNamespace(yaml, namespace string) error {
	originalNamespace := s.kubectlOptions.Namespace
	s.kubectlOptions.Namespace = namespace
	defer func() {
		s.kubectlOptions.Namespace = originalNamespace
	}()
	s.addFinalizers(func() {
		_ = s.DeleteResourceFromStringWithNamespace(yaml, namespace)
	})
	return s.CreateResourceFromString(yaml)
}

func (s *Scaffold) DeleteResourceFromStringWithNamespace(yaml, namespace string) error {
	originalNamespace := s.kubectlOptions.Namespace
	s.kubectlOptions.Namespace = namespace
	defer func() {
		s.kubectlOptions.Namespace = originalNamespace
	}()
	return k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
}

func (s *Scaffold) NewAPISIX() (dashboard.Dashboard, error) {
	return dashboard.NewClient()
}

func (s *Scaffold) ClusterClient() (apisix.Cluster, error) {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:7080",
		Path:   "/apisix/admin",
	}
	cli, err := s.NewAPISIX()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(context.Background(), &apisix.ClusterOptions{
		BaseURL:        u.String(),
		ControllerName: s.opts.ControllerName,
		Labels:         map[string]string{"k8s/controller-name": s.opts.ControllerName},
		AdminKey:       s.opts.APISIXAdminAPIKey,
		SyncCache:      true,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster(""), nil
}

func (s *Scaffold) shutdownApisixTunnel() {
	s.apisixHttpTunnel.Close()
	s.apisixHttpsTunnel.Close()
}

// Namespace returns the current working namespace.
func (s *Scaffold) Namespace() string {
	return s.kubectlOptions.Namespace
}

func (s *Scaffold) EnsureNumEndpointsReady(t testing.TestingT, endpointsName string, desired int) {
	e, err := k8s.GetKubernetesClientFromOptionsE(t, s.kubectlOptions)
	Expect(err).ToNot(HaveOccurred(), "Getting Kubernetes clientset")

	statusMsg := fmt.Sprintf("Wait for endpoints %s to be ready.", endpointsName)
	message := retry.DoWithRetry(
		t,
		statusMsg,
		20,
		2*time.Second,
		func() (string, error) {
			endpoints, err := e.CoreV1().Endpoints(s.Namespace()).Get(context.Background(), endpointsName, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			readyNum := 0
			for _, subset := range endpoints.Subsets {
				readyNum += len(subset.Addresses)
			}
			if readyNum == desired {
				return "Service is now available", nil
			}
			return "failed", fmt.Errorf("endpoints not ready yet, expect %v, actual %v", desired, readyNum)
		},
	)
	GinkgoT().Log(message)
}

// GetKubernetesClient get kubernetes client use by scaffold
func (s *Scaffold) GetKubernetesClient() *kubernetes.Clientset {
	client, err := k8s.GetKubernetesClientFromOptionsE(s.t, s.kubectlOptions)
	Expect(err).ToNot(HaveOccurred(), "Getting Kubernetes clientset")
	return client
}

func (s *Scaffold) RunKubectlAndGetOutput(args ...string) (string, error) {
	return k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, args...)
}

func (s *Scaffold) RunDigDNSClientFromK8s(args ...string) (string, error) {
	kubectlArgs := []string{
		"run",
		"dig",
		"-i",
		"--rm",
		"--restart=Never",
		"--image-pull-policy=IfNotPresent",
		"--image=toolbelt/dig",
		"--",
	}
	kubectlArgs = append(kubectlArgs, args...)
	return s.RunKubectlAndGetOutput(kubectlArgs...)
}
