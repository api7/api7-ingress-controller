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
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/gavv/httpexpect/v2"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

var (
	valuesTemplate *template.Template
	_db            string
)

func init() {
	_db = os.Getenv("DB")
	if _db == "" {
		_db = postgres
	}
	tmpl, err := template.New("values.yaml").Parse(`
dashboard:
  image:
    repository: hkccr.ccs.tencentyun.com/api7-dev/api7-ee-3-integrated
    pullPolicy: IfNotPresent
    tag: {{ .Tag }}
  extraEnvVars:
    - name: GOCOVERDIR
      value: /app/covdatafiles
  extraVolumes:
    - name: cover
      hostPath:
        path: /tmp/covdatafiles
        type: DirectoryOrCreate
  extraVolumeMounts:
    - name: cover
      mountPath: /app/covdatafiles
dp_manager:
  image:
    repository: hkccr.ccs.tencentyun.com/api7-dev/api7-ee-dp-manager
    pullPolicy: IfNotPresent
    tag: {{ .Tag }}
  extraEnvVars:
    - name: GOCOVERDIR
      value: /app/covdatafiles
  extraVolumes:
    - name: cover
      hostPath:
        path: /tmp/covdatafiles
        type: DirectoryOrCreate
  extraVolumeMounts:
    - name: cover
      mountPath: /app/covdatafiles
fullnameOverride: api7ee3
podSecurityContext:
  runAsUser: 0
dashboard_configuration:
  log:
    level: debug
  database:
    dsn: {{ .DSN }}
  server:
    listen:
      disable: false
      host: "0.0.0.0"
      port: 7080
    tls:
      disable: false
      host: "0.0.0.0"
      port: 7443
    status:
      host: "0.0.0.0"
    cron_spec: "@every 1s"
  plugins:
    - error-page
    - real-ip
    #- ai
    - error-page
    - client-control
    - proxy-control
    - zipkin
    - skywalking
    - ext-plugin-pre-req
    - mocking
    - serverless-pre-function
    - batch-requests
    - ua-restriction
    - referer-restriction
    - uri-blocker
    - request-validation
    - authz-casbin
    - authz-casdoor
    - wolf-rbac
    - multi-auth
    - ldap-auth
    - forward-auth
    - saml-auth
    - opa
    - authz-keycloak
    #- error-log-logger
    - proxy-mirror
    - proxy-cache
    - api-breaker
    - limit-req
    #- node-status
    - gzip
    - kafka-proxy
    #- dubbo-proxy
    - grpc-transcode
    - grpc-web
    - public-api
    - data-mask
    - opentelemetry
    - datadog
    - echo
    - loggly
    - splunk-hec-logging
    - skywalking-logger
    - google-cloud-logging
    - sls-logger
    - tcp-logger
    - rocketmq-logger
    - udp-logger
    - file-logger
    - clickhouse-logger
    - ext-plugin-post-resp
    - serverless-post-function
    - azure-functions
    - aws-lambda
    - openwhisk
    - consumer-restriction
    - acl
    - basic-auth
    - cors
    - csrf
    - fault-injection
    - hmac-auth
    - jwt-auth
    - key-auth
    - openid-connect
    - limit-count
    - redirect
    - request-id
    - proxy-rewrite
    - response-rewrite
    - workflow
    - proxy-buffering
    - tencent-cloud-cls
    - openfunction
    - graphql-proxy-cache
    - ext-plugin-post-req
    #- log-rotate
    - graphql-limit-count
    - elasticsearch-logger
    - kafka-logger
    - body-transformer
    - traffic-split
    - degraphql
    - http-logger
    - cas-auth
    - traffic-label
    - oas-validator
    - api7-traffic-split
    - limit-conn
    - prometheus
    - syslog
    - ip-restriction
dp_manager_configuration:
  api_call_flush_period: 1s
  server:
    status:
      host: "0.0.0.0"
  log:
    level: debug
  database:
    dsn: {{ .DSN }}
prometheus:
  builtin: false
  server:
    persistence:
      enabled: false
postgresql:
{{- if ne .DB "postgres" }}
  builtin: false
{{- end }}
  primary:
    containerSecurityContext:
      enabled: false
    persistence:
      enabled: false
  readReplicas:
    persistence:
      enabled: false
developer_portal_configuration:
  enable: false
dashboard_service:
  type: ClusterIP
  annotations: {}
  port: 7080
  tlsPort: 7443
  ingress:
    enabled: false
    className: ""
    annotations: {}
      # kubernetes.io/ingress.class: nginx
      # kubernetes.io/tls-acme: "true"
    hosts:
      - host: dashboard.local
        paths:
          - path: /
            pathType: ImplementationSpecific
            # backend:
            #   service:
            #     name: api7ee3-dashboard
            #     port:
            #       number: 7943
    tls: []
api_usage:
  service:
    ingress:
      enabled: false
`)
	if err != nil {
		panic(err)
	}
	valuesTemplate = tmpl
}

// DatabaseConfig is the database related configuration entrypoint.
type DatabaseConfig struct {
	DSN string `json:"dsn" yaml:"dsn" mapstructure:"dsn"`

	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
}

type LogOptions struct {
	// Level is the minimum logging level that a logging message should have
	// to output itself.
	Level string `json:"level" yaml:"level"`
	// Output defines the destination file path to output logging messages.
	// Two keywords "stderr" and "stdout" can be specified so that message will
	// be written to stderr or stdout.
	Output string `json:"output" yaml:"output"`
}

func (conf *DatabaseConfig) GetType() string {
	parts := strings.SplitN(conf.DSN, "://", 2)
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}

//nolint:unused
func getDSN() string {
	switch _db {
	case postgres:
		return postgresDSN
	case oceanbase:
		return oceanbaseDSN
	case mysql:
		return mysqlDSN
	}
	panic("unknown database")
}

type responseCreateGateway struct {
	Value    responseCreateGatewayValue `json:"value"`
	ErrorMsg string                     `json:"error_msg"`
}

type responseCreateGatewayValue struct {
	ID             string `json:"id"`
	TokenPlainText string `json:"token_plain_text"`
	Key            string `json:"key"`
}

type BaseCertificate struct {
	ID          string `json:"id" gorm:"primaryKey; column:id; size:255;"`
	Certificate string `json:"certificate" gorm:"column:certificate; type:text;"`
	PrivateKey  string `json:"private_key" gorm:"column:private_key; type:text;" mask:"fixed"`
}

type DataplaneCertificate struct {
	*BaseCertificate

	GatewayGroupID string `json:"gateway_group_id" gorm:"column:gateway_group_id;size:255;"`
	CACertificate  string `json:"ca_certificate" gorm:"column:ca_certificate;type:text;"`
}

func (DataplaneCertificate) TableName() string {
	return "dataplane_certificate"
}

func (f *Framework) GetDataplaneCertificates(gatewayGroupID string) *DataplaneCertificate {
	respExp := f.DashboardHTTPClient().
		POST("/api/gateway_groups/"+gatewayGroupID+"/dp_client_certificates").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes([]byte(`{}`)).
		Expect()

	f.Logger.Logf(f.GinkgoT, "dataplane certificates issuer response: %s", respExp.Body().Raw())

	respExp.Status(200).Body().Contains("certificate").Contains("private_key").Contains("ca_certificate")
	body := respExp.Body().Raw()

	var dpCertResp struct {
		Value DataplaneCertificate `json:"value"`
	}
	err := json.Unmarshal([]byte(body), &dpCertResp)
	Expect(err).ToNot(HaveOccurred())

	return &dpCertResp.Value
}

func (s *Framework) GetAdminKey(gatewayGroupID string) string {
	respExp := s.DashboardHTTPClient().PUT("/api/gateway_groups/"+gatewayGroupID+"/admin_key").
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	respExp.Status(200).Body().Contains("key")

	body := respExp.Body().Raw()

	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	Expect(err).ToNot(HaveOccurred(), "unmarshal response")
	return response.Value.Key
}

func (f *Framework) DeleteGatewayGroup(gatewayGroupID string) {
	respExp := f.DashboardHTTPClient().
		DELETE("/api/gateway_groups/"+gatewayGroupID).
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	body := respExp.Body().Raw()

	// unmarshal into responseCreateGateway
	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	Expect(err).ToNot(HaveOccurred())
}

func (f *Framework) CreateNewGatewayGroupWithIngress() string {
	gid, err := f.CreateNewGatewayGroupWithIngressE()
	Expect(err).ToNot(HaveOccurred())
	return gid
}

func (f *Framework) CreateNewGatewayGroupWithIngressE() (string, error) {
	gatewayGroupName := uuid.NewString()
	payload := []byte(fmt.Sprintf(
		`{"name":"%s","description":"","labels":{},"type":"api7_ingress_controller"}`,
		gatewayGroupName,
	))

	respExp := f.DashboardHTTPClient().
		POST("/api/gateway_groups").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payload).
		Expect()

	f.Logger.Logf(f.GinkgoT, "create gateway group response: %s", respExp.Body().Raw())

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

func (f *Framework) setDpManagerEndpoints() {
	payload := []byte(fmt.Sprintf(`{"dp_manager_address":["%s"]}`, DPManagerTLSEndpoint))

	respExp := f.DashboardHTTPClient().
		PUT("/api/system_settings").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payload).
		Expect()

	respExp.Raw()
	f.Logf("set dp manager endpoints response: %s", respExp.Body().Raw())

	respExp.Status(200).
		Body().Contains("dp_manager_address")
}

func (f *Framework) GetDashboardEndpoint() string {
	return _dashboardHTTPTunnel.Endpoint()
}

func (f *Framework) GetDashboardEndpointHTTPS() string {
	return _dashboardHTTPSTunnel.Endpoint()
}

func (f *Framework) DashboardHTTPClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   f.GetDashboardEndpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(f.GinkgoT),
		),
	})
}

func (f *Framework) DashboardHTTPSClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   f.GetDashboardEndpointHTTPS(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(f.GinkgoT),
		),
	})
}
