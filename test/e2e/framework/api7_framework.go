// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultDashboardVersion = "dev"

var (
	API7EELicense string

	dashboardVersion string
)

// initSuiteEnv reads required environment variables and panics early with a clear
// message if any mandatory variable is missing.
func initSuiteEnv() {
	API7EELicense = os.Getenv("API7_EE_LICENSE")
	if API7EELicense == "" {
		panic("env {API7_EE_LICENSE} is required")
	}

	dashboardVersion = os.Getenv("DASHBOARD_VERSION")
	if dashboardVersion == "" {
		dashboardVersion = defaultDashboardVersion
	}
}

func (f *Framework) BeforeSuite() {
	initSuiteEnv()

	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)

	Eventually(func() error {
		_, err := k8s.GetNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("namespace %s still exists", _namespace)
	}, "1m", "2s").Should(Succeed())

	k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, _namespace)

	initialPassword := f.deployComponents()

	time.Sleep(1 * time.Minute)
	err := f.newDashboardTunnel()
	f.Logf("Dashboard HTTP Tunnel:" + _dashboardHTTPTunnel.Endpoint())
	Expect(err).ShouldNot(HaveOccurred(), "creating dashboard tunnel")

	f.rotateDashboardPassword(initialPassword)
	f.UploadLicense()

	f.setDpManagerEndpoints()
}

func (f *Framework) AfterSuite() {
	f.shutdownDashboardTunnel()
}

// DeployAPI7EE deploys the API7EE control plane once (runs on ginkgo node 1 only).
// It passes the rotated administrator password to InitNodeConnections on all nodes.
func (f *Framework) DeployAPI7EE() []byte {
	initSuiteEnv()

	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)

	Eventually(func() error {
		_, err := k8s.GetNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("namespace %s still exists", _namespace)
	}, "1m", "2s").Should(Succeed())

	k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, _namespace)

	initialPassword := f.deployComponents()

	time.Sleep(1 * time.Minute)

	// Create a temporary tunnel for one-time setup operations.
	// Each node will create its own persistent tunnel in InitNodeConnections.
	err := f.newDashboardTunnel()
	Expect(err).ShouldNot(HaveOccurred(), "creating temporary dashboard tunnel")
	f.Logf("Temporary dashboard tunnel: %s", _dashboardHTTPTunnel.Endpoint())

	f.rotateDashboardPassword(initialPassword)
	f.UploadLicense()
	f.setDpManagerEndpoints()

	// Close the temporary tunnel; each node creates its own in InitNodeConnections.
	f.shutdownDashboardTunnel()

	return []byte(f.dashboardAdminPassword)
}

// InitNodeConnections initializes per-node connections to the shared API7EE control plane.
// It runs on every ginkgo parallel node after DeployAPI7EE completes.
func (f *Framework) InitNodeConnections(adminPassword []byte) {
	initSuiteEnv()
	f.dashboardAdminPassword = string(adminPassword)

	err := f.newDashboardTunnel()
	Expect(err).ShouldNot(HaveOccurred(), "creating dashboard tunnel for node")
	f.Logf("Dashboard HTTP Tunnel: %s", _dashboardHTTPTunnel.Endpoint())
}

// CloseNodeConnections closes per-node connections. Runs on every ginkgo parallel node.
func (f *Framework) CloseNodeConnections() {
	f.shutdownDashboardTunnel()
}

// TeardownInfrastructure cleans up suite-level resources. Runs on ginkgo node 1 only.
// The Kind cluster is deleted by CI after the job, so this is a no-op.
func (f *Framework) TeardownInfrastructure() {}

func (f *Framework) deployComponents() string {
	initialPassword := newDashboardPassword()
	_, err := f.clientset.CoreV1().Secrets(_namespace).Create(f.Context, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dashboardInitialPasswordSecret},
		StringData: map[string]string{"password": initialPassword},
	}, metav1.CreateOptions{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating dashboard initial password secret")

	f.deploy()
	f.initDashboard()
	return initialPassword
}

func (f *Framework) UploadLicense() {
	payload := map[string]any{"data": API7EELicense}
	payloadBytes, err := json.Marshal(payload)
	assert.Nil(f.GinkgoT, err)

	respExpect := f.DashboardHTTPClient().PUT("/api/license").
		WithHeader("Content-Type", "application/json").
		WithBytes(payloadBytes).
		Expect()

	respExpect.Status(200)
}

func (f *Framework) deploy() {
	debug := func(format string, v ...any) {
		log.Infof(format, v...)
	}

	kubeConfigPath := os.Getenv("KUBECONFIG")
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(
		kube.GetConfig(kubeConfigPath, "", f.kubectlOpts.Namespace),
		f.kubectlOpts.Namespace,
		"memory",
		debug,
	)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "init helm action config")

	install := action.NewInstall(actionConfig)
	install.Namespace = f.kubectlOpts.Namespace
	install.ReleaseName = "api7ee3"

	chartPath, err := install.LocateChart("api7/api7ee3", cli.New())
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "locate helm chart")

	chart, err := loader.Load(chartPath)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "load helm chart")

	buf := bytes.NewBuffer(nil)
	_ = valuesTemplate.Execute(buf, map[string]any{
		"DB":                         _db,
		"DSN":                        getDSN(),
		"Tag":                        dashboardVersion,
		"AdminInitialPasswordSecret": dashboardInitialPasswordSecret,
	})

	f.Logf("values: %s", buf.String())

	var v map[string]any
	err = yaml.Unmarshal(buf.Bytes(), &v)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "unmarshal values")
	_, err = install.Run(chart, v)
	if err != nil {
		f.Logf("install dashboard failed, err: %v", err)
	}
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "install dashboard")

	err = f.ensureService("api7ee3-dashboard", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring dashboard service")

	err = f.ensureService("api7-postgresql", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring postgres service")
}

func (f *Framework) initDashboard() {
	f.deletePods("app.kubernetes.io/name=api7ee3")
	time.Sleep(5 * time.Second)
}

var (
	_dashboardHTTPTunnel  *k8s.Tunnel
	_dashboardHTTPSTunnel *k8s.Tunnel
)

// dashboardLocalPorts returns the local port pair to use for the dashboard HTTP
// and HTTPS tunnels.  Each ginkgo parallel process gets a unique, non-overlapping
// range based on its 1-indexed process number, eliminating port conflicts without
// any TOCTOU race.
//
// Formula: base = 18000 + node*100
//
//	node=1 → 18100 (HTTP) / 18101 (HTTPS)
//	node=2 → 18200 (HTTP) / 18201 (HTTPS)
func dashboardLocalPorts() (httpLocal, httpsLocal int) {
	node := GinkgoParallelProcess() // 1-indexed
	base := 18000 + node*100
	return base, base + 1
}

func (f *Framework) newDashboardTunnel() error {
	var (
		httpPort  int
		httpsPort int
	)

	service := k8s.GetService(f.GinkgoT, f.kubectlOpts, "api7ee3-dashboard")

	for _, port := range service.Spec.Ports {
		switch port.Name {
		case "http":
			httpPort = int(port.Port)
		case "https":
			httpsPort = int(port.Port)
		}
	}

	httpLocal, httpsLocal := dashboardLocalPorts()
	_dashboardHTTPTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpLocal, httpPort)
	_dashboardHTTPSTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpsLocal, httpsPort)

	if err := _dashboardHTTPTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}
	if err := _dashboardHTTPSTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}

	return nil
}

func (f *Framework) shutdownDashboardTunnel() {
	if _dashboardHTTPTunnel != nil {
		_dashboardHTTPTunnel.Close()
	}
	if _dashboardHTTPSTunnel != nil {
		_dashboardHTTPSTunnel.Close()
	}
}
