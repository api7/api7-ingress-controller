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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"

	v1 "github.com/api7/api7-ingress/api/apisix/v1"
	"github.com/api7/api7-ingress/test/e2e/framework"
)

type responseCreateGateway struct {
	Value    responseCreateGatewayValue `json:"value"`
	ErrorMsg string                     `json:"error_msg"`
}

func (s *Scaffold) DeployDataplaneWithIngress() {
	s.deploy()

	err := s.waitAllAPISIXPodsAvailable()
	assert.Nil(s.t, err, "waiting for apisix ready")

	time.Sleep(10 * time.Second)

	err = s.newAPISIXTunnels()
	assert.Nil(s.t, err, "creating apisix tunnels")
}

type responseCreateGatewayValue struct {
	ID             string `json:"id"`
	TokenPlainText string `json:"token_plain_text"`
	Key            string `json:"key"`
}

func (s *Scaffold) GetAPIKey() (string, error) {
	gatewayGroupID := s.gatewaygroupid

	respExp := s.DashboardHTTPClient().PUT("/api/gateway_groups/"+gatewayGroupID+"/admin_key").
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	respExp.Status(200).Body().Contains("key")

	body := respExp.Body().Raw()

	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		return "", err
	}
	return response.Value.Key, nil
}

func (s *Scaffold) DeleteGatewayGroup() {
	gatewayGroupID := s.gatewaygroupid

	respExp := s.DashboardHTTPClient().
		DELETE("/api/gateway_groups/"+gatewayGroupID).
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	body := respExp.Body().Raw()

	//unmarshal into responseCreateGateway
	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	Expect(err).To(BeNil())
}

func (s *Scaffold) CreateNewGatewayGroupWithIngress() string {
	gid, err := s.CreateNewGatewayGroupWithIngressE()
	Expect(err).To(BeNil())
	return gid
}

func (s *Scaffold) CreateNewGatewayGroupWithIngressE() (string, error) {
	gatewayGroupName := uuid.NewString()
	payload := []byte(fmt.Sprintf(`{"name":"%s","description":"","labels":{},"type":"api7_ingress_controller"}`, gatewayGroupName))

	respExp := s.DashboardHTTPClient().
		POST("/api/gateway_groups").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payload).
		Expect()

	s.Logger.Logf(s.t, "create gateway group response: %s", respExp.Body().Raw())

	respExp.Status(200).Body().Contains("id")

	body := respExp.Body().Raw()

	var response responseCreateGateway

	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		return "", err
	}

	if response.ErrorMsg != "" {
		return "", fmt.Errorf("error creating gateway group: %s", response.ErrorMsg)
	}
	return response.Value.ID, nil
}

func (s *Scaffold) getTokenFromDashboard() (string, error) {
	gatewayGroupID := s.gatewaygroupid

	respExp := s.DashboardHTTPClient().
		POST("/api/gateway_groups/"+gatewayGroupID+"/instance_token").
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	respExp.Status(200).Body().Contains("token_plain_text")
	body := respExp.Body().Raw()
	//unmarshal into responseCreateGateway
	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		return "", err
	}
	return response.Value.TokenPlainText, nil
}

func indent(data string) string {
	list := strings.Split(data, "\n")
	for i := 0; i < len(list); i++ {
		list[i] = "    " + list[i]
	}
	return strings.Join(list, "\n")
}

func (s *Scaffold) waitAllAPISIXPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=apisix",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no apisix pods created")
			return false, nil
		}
		for _, item := range items {
			foundPodReady := false
			for _, cond := range item.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				foundPodReady = true
				if cond.Status != "True" {
					return false, nil
				}
			}
			if !foundPodReady {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

func (f *Scaffold) applySSLSecret(name string, cert, pkey, caCert []byte) {
	kind := "Secret"
	apiVersion := "v1"
	secretType := corev1.SecretTypeTLS
	secret := applycorev1.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       &kind,
			APIVersion: &apiVersion,
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: &name,
		},
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": pkey,
			"ca.crt":  caCert,
		},
		Type: &secretType,
	}

	cli, err := k8s.GetKubernetesClientE(ginkgo.GinkgoT())

	_, err = cli.CoreV1().Secrets(f.Namespace()).Apply(context.TODO(), &secret, metav1.ApplyOptions{
		FieldManager: "e2e",
	})
	f.GomegaT.Expect(err).Should(BeNil(), "apply TLS secret")
}

func (s *Scaffold) GetDataplaneCertificates() *v1.DataplaneCertificate {
	respExp := s.DashboardHTTPClient().
		POST("/api/gateway_groups/"+s.gatewaygroupid+"/dp_client_certificates").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		Expect()

	s.Logger.Logf(ginkgo.GinkgoT(), "dataplane certificates issuer response: %s", respExp.Body().Raw())

	respExp.Status(200).Body().Contains("certificate").Contains("private_key").Contains("ca_certificate")
	body := respExp.Body().Raw()

	var dpCertResp struct {
		Value v1.DataplaneCertificate `json:"value"`
	}
	err := json.Unmarshal([]byte(body), &dpCertResp)
	Expect(err).To(BeNil())

	return &dpCertResp.Value
}

func (s *Scaffold) deploy() {
	dpCert := s.GetDataplaneCertificates()

	s.applySSLSecret("dp-ssl", []byte(dpCert.Certificate), []byte(dpCert.PrivateKey), []byte(dpCert.CACertificate))

	buf := bytes.NewBuffer(nil)

	_ = framework.DPSpecTpl.Execute(buf, map[string]any{
		"TLSEnabled":             true,
		"DPManagerEndpoint":      framework.DPManagerTLSEndpoint,
		"SetEnv":                 true,
		"SSLKey":                 framework.TestKey,
		"SSLCert":                framework.TestCert,
		"ForIngressGatewayGroup": true,
	})

	k8s.KubectlApplyFromString(s.t, s.kubectlOptions, buf.String())
}
