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

package webhook

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Gateway Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "gateway-webhook-test",
		EnableWebhook: true,
	})

	Context("GatewayProxy reference validation warnings", func() {
		It("should warn when referenced GatewayProxy does not exist on create and update", func() {
			By("creating GatewayClass with controller name")
			err := s.CreateResourceFromString(s.GetGatewayClassYaml())
			Expect(err).ShouldNot(HaveOccurred())

			time.Sleep(2 * time.Second)

			By("creating Gateway referencing a missing GatewayProxy")
			missingName := "missing-proxy"
			gwYAML := `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: http1
    protocol: HTTP
    port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: %s
`

			output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), missingName))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName)))

			time.Sleep(2 * time.Second)

			By("updating Gateway to reference another missing GatewayProxy")
			missingName2 := "missing-proxy-2"
			output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), missingName2))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName2)))

			By("create GatewayProxy")
			err = s.CreateResourceFromString(s.GetGatewayProxySpec())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("updating Gateway to reference an existing GatewayProxy")
			existingName := "apisix-proxy-config"
			output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), existingName))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), existingName)))

			By("delete Gateway")
			err = s.DeleteResource("Gateway", s.Namespace())
			Expect(err).ShouldNot(HaveOccurred())

			By("delete GatewayClass")
			err = s.DeleteResource("GatewayClass", s.Namespace())
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("GatewayProxy configuration conflicts", func() {
		It("should reject GatewayProxy that reuses the same Service and AdminKey Secret as an existing one on create and update", func() {
			serviceTemplate := `
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: dummy-control-plane
  ports:
  - name: admin
    port: 9180
    targetPort: 9180
`
			secretTemplate := `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: Opaque
stringData:
  %s: %s
`
			gatewayProxyTemplate := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          valueFrom:
            secretKeyRef:
              name: %s
              key: token
`

			serviceName := "gatewayproxy-shared-service"
			secretName := "gatewayproxy-shared-secret"
			initialProxy := "gatewayproxy-shared-primary"
			conflictingProxy := "gatewayproxy-shared-conflict"

			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, serviceName))).ShouldNot(HaveOccurred(), "creating shared Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(secretTemplate, secretName, "token", "value"))).ShouldNot(HaveOccurred(), "creating shared Secret")

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, initialProxy, serviceName, secretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating initial GatewayProxy")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, conflictingProxy, serviceName, secretName))
			Expect(err).Should(HaveOccurred(), "expecting conflict for duplicated GatewayProxy")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), conflictingProxy)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), initialProxy)))
			Expect(err.Error()).To(ContainSubstring("Service"))
			Expect(err.Error()).To(ContainSubstring("AdminKey secret"))

			Expect(s.DeleteResource("GatewayProxy", initialProxy)).ShouldNot(HaveOccurred())
			Expect(s.DeleteResource("Service", serviceName)).ShouldNot(HaveOccurred())
			Expect(s.DeleteResource("Secret", secretName)).ShouldNot(HaveOccurred())
		})

		It("should reject GatewayProxy that overlaps endpoints when sharing inline AdminKey value", func() {
			gatewayProxyTemplate := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

			existingProxy := "gatewayproxy-inline-primary"
			conflictingProxy := "gatewayproxy-inline-conflict"
			endpointA := "https://127.0.0.1:9443"
			endpointB := "https://10.0.0.1:9443"
			endpointC := "https://192.168.0.1:9443"
			inlineKey := "inline-credential"

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, existingProxy, endpointA, endpointB, inlineKey))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy with inline AdminKey")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, conflictingProxy, endpointB, endpointC, inlineKey))
			Expect(err).Should(HaveOccurred(), "expecting conflict for overlapping endpoints with shared AdminKey")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), conflictingProxy)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), existingProxy)))
			Expect(err.Error()).To(ContainSubstring("control plane endpoints"))
			Expect(err.Error()).To(ContainSubstring("inline AdminKey value"))
		})

		It("should reject GatewayProxy update that creates conflict with another GatewayProxy", func() {
			serviceTemplate := `
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: dummy-control-plane
  ports:
  - name: admin
    port: 9180
    targetPort: 9180
`
			secretTemplate := `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: Opaque
stringData:
  %s: %s
`
			gatewayProxyTemplate := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          valueFrom:
            secretKeyRef:
              name: %s
              key: token
`

			sharedServiceName := "gatewayproxy-update-shared-service"
			sharedSecretName := "gatewayproxy-update-shared-secret"
			uniqueServiceName := "gatewayproxy-update-unique-service"
			proxyA := "gatewayproxy-update-a"
			proxyB := "gatewayproxy-update-b"

			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, sharedServiceName))).ShouldNot(HaveOccurred(), "creating shared Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, uniqueServiceName))).ShouldNot(HaveOccurred(), "creating unique Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(secretTemplate, sharedSecretName, "token", "value"))).ShouldNot(HaveOccurred(), "creating shared Secret")

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyA, sharedServiceName, sharedSecretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy A with shared Service and Secret")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyB, uniqueServiceName, sharedSecretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy B with unique Service but same Secret")

			time.Sleep(2 * time.Second)

			By("updating GatewayProxy B to use the same Service as GatewayProxy A, causing conflict")
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyB, sharedServiceName, sharedSecretName))
			Expect(err).Should(HaveOccurred(), "expecting conflict when updating to same Service")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), proxyA)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), proxyB)))
		})
	})
})
