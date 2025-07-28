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
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega" //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed manifests/dp.yaml
	_dpSpec   string
	DPSpecTpl *template.Template
)

func init() {
	tpl, err := template.New("dp").Funcs(sprig.TxtFuncMap()).Parse(_dpSpec)
	if err != nil {
		panic(err)
	}
	DPSpecTpl = tpl
}

type API7DeployOptions struct {
	Namespace string
	Name      string

	GatewayGroupID         string
	TLSEnabled             bool
	SSLKey                 string
	SSLCert                string
	DPManagerEndpoint      string
	SetEnv                 bool
	ForIngressGatewayGroup bool

	ServiceName      string
	ServiceType      string
	ServiceHTTPPort  int
	ServiceHTTPSPort int
	Replicas         *int
}

func (f *Framework) DeployGateway(opts *API7DeployOptions) *corev1.Service {
	if opts.ServiceName == "" {
		opts.ServiceName = "api7ee3-apisix-gateway-mtls"
	}

	if opts.ServiceHTTPPort == 0 {
		opts.ServiceHTTPPort = 80
	}

	if opts.ServiceHTTPSPort == 0 {
		opts.ServiceHTTPSPort = 443
	}

	dpCert := f.GetDataplaneCertificates(opts.GatewayGroupID)

	f.applySSLSecret(opts.Namespace,
		"dp-ssl",
		[]byte(dpCert.Certificate),
		[]byte(dpCert.PrivateKey),
		[]byte(dpCert.CACertificate),
	)

	buf := bytes.NewBuffer(nil)

	_ = DPSpecTpl.Execute(buf, opts)

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err := WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=apisix",
	})
	Expect(err).ToNot(HaveOccurred(), "waiting for gateway pod ready")

	Eventually(func() bool {
		svc, err := k8s.GetServiceE(f.GinkgoT, kubectlOpts, opts.ServiceName)
		if err != nil {
			f.Logf("failed to get service %s: %v", opts.ServiceName, err)
			return false
		}
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			return len(svc.Status.LoadBalancer.Ingress) > 0
		}
		return true
	}, "20s", "4s").Should(BeTrue(), "waiting for LoadBalancer IP")

	svc, err := k8s.GetServiceE(f.GinkgoT, kubectlOpts, opts.ServiceName)
	Expect(err).ToNot(HaveOccurred(), "failed to get service %s: %v", opts.ServiceName, err)
	return svc
}
