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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_apisixConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: apisix-gw-config.yaml
data:
  config.yaml: |
%s
`

	_eeService = `
  apiVersion: v1
  kind: Service
  metadata:
    name: api7-ee-gateway-1
  spec:
    selector:
      app: api7-ee-gateway-1
    ports:
      - protocol: TCP
        name: http
        port: 9080
        targetPort: 9080
      - protocol: TCP
        name: https
        port: 9443
        targetPort: 9443  
  `

	_eeDeployment = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: api7-ee-gateway-1
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: api7-ee-gateway-1
    template:
      metadata:
        labels:
          app: api7-ee-gateway-1
      spec:
        containers:
          - name: api7-ee-gateway-1
            image: localhost:5000/hkccr.ccs.tencentyun.com/api7-dev/api7-ee-3-gateway:dev
            ports:
              - containerPort: 9080
              - containerPort: 9443
            env:
              - name: API7_CONTROL_PLANE_ENDPOINTS
                value: '["http://dp-manager:7900"]'
              - name: API7_CONTROL_PLANE_TOKEN
                value: %s
            volumeMounts:
              - name: config-volume
                mountPath: /usr/local/apisix/conf
                readOnly: true
        volumes:
          - name: config-volume
            hostPath:
              path: ./gateway_conf
`
)

type APISIXConfig struct {
	// Used for template rendering.
	EtcdServiceFQDN string
}

func (s *Scaffold) newAPISIXConfigMap(cm *APISIXConfig) error {
	if cm == nil {
		return fmt.Errorf("config not allowed to be empty")
	}
	data, err := s.renderConfig(s.opts.APISIXConfigPath, cm)
	if err != nil {
		return err
	}
	data = indent(data)
	configData := fmt.Sprintf(_apisixConfigMap, data)
	if err := s.CreateResourceFromString(configData); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) UploadLicense() error {
	payload := []byte(fmt.Sprintf(`{"data":"%s"}`, tenyearsLicense))
	url := fmt.Sprintf("http://%s:%d/api/license", DashboardHost, DashboardPort)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("THE ERROR IS ", err.Error())
		return err
	}
	defer resp.Body.Close()
	return nil
}

type responseCreateGateway struct {
	Value responseCreateGatewayValue `json:"value"`
}
type responseCreateGatewayValue struct {
	ID             string `json:"id"`
	TokenPlainText string `json:"token_plain_text"`
	Key            string `json:"key"`
	ErrorMsg       string `json:"error_msg"`
}

func (s *Scaffold) GetAPIKey() (string, error) {
	gatewayGroupID := s.gatewaygroupid
	url := fmt.Sprintf("http://%s:%d/api/gateway_groups/%s/admin_key", DashboardHost, DashboardPort, gatewayGroupID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Set basic authentication
	req.SetBasicAuth("admin", "admin")

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	//unmarshal into responseCreateGateway
	var response responseCreateGateway
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	if response.Value.ErrorMsg != "" {
		return "", fmt.Errorf("error getting key: %s", response.Value.ErrorMsg)
	}
	return response.Value.Key, nil
}

func (s *Scaffold) DeleteGatewayGroup() error {
	gatewayGroupID := s.gatewaygroupid
	url := fmt.Sprintf("http://%s:%d/api/gateway_groups/%s", DashboardHost, DashboardPort, gatewayGroupID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Set basic authentication
	req.SetBasicAuth("admin", "admin")

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Scaffold) CreateNewGatewayGroup() (string, error) {
	payload := []byte(`{"name":"ingress10","description":"","labels":{},"type":"api7_ingress_controller"}`)
	url := fmt.Sprintf("http://%s:%d/api/gateway_groups", DashboardHost, DashboardPort)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Set basic authentication
	req.SetBasicAuth("admin", "admin")

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	//unmarshal into responseCreateGateway
	var response responseCreateGateway
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	if response.Value.ErrorMsg != "" {
		return "", fmt.Errorf("error creating gateway group: %s", response.Value.ErrorMsg)
	}
	fmt.Println("GOT THE ID ", response.Value.ID)
	return response.Value.ID, nil
}

func (s *Scaffold) getTokenFromDashboard() (string, error) {
	gatewayGroupID := s.gatewaygroupid
	url := fmt.Sprintf("http://%s:%d/api/gateway_groups/%s/instance_token", DashboardHost, DashboardPort, gatewayGroupID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Set basic authentication
	req.SetBasicAuth("admin", "admin")

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	//unmarshal into responseCreateGateway
	var response responseCreateGateway
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	return response.Value.TokenPlainText, nil
}

func (s *Scaffold) newDataplane() (*corev1.Service, error) {
	token, err := s.getTokenFromDashboard()
	if err != nil {
		return nil, err
	}
	if err := s.CreateResourceFromString(fmt.Sprintf(_eeDeployment, token)); err != nil {
		return nil, err
	}
	if err := s.CreateResourceFromString(_eeService); err != nil {
		return nil, err
	}
	fmt.Println("Service is ", _eeService)
	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "api7-ee-gateway-1")
	if err != nil {
		return nil, err
	}

	return svc, nil

}

// func (s *Scaffold) newAPISIX() (*corev1.Service, error) {
// 	deployment := fmt.Sprintf(_apisixDeployment, EtcdServiceName)
// 	if err := s.CreateResourceFromString(
// 		s.FormatRegistry(deployment),
// 	); err != nil {
// 		return nil, err
// 	}
// 	if err := s.CreateResourceFromString(_apisixService); err != nil {
// 		return nil, err
// 	}

// 	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "apisix-service-e2e-test")
// 	if err != nil {
// 		return nil, err
// 	}

// 	return svc, nil
// }

func indent(data string) string {
	list := strings.Split(data, "\n")
	for i := 0; i < len(list); i++ {
		list[i] = "    " + list[i]
	}
	return strings.Join(list, "\n")
}

func (s *Scaffold) waitAllAPISIXPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=api7-ee-gateway-1",
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
