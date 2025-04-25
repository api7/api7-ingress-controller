package framework

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/gopkg/pkg/log"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"golang.org/x/net/html"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	API7EELicense string

	valuesTemplate *template.Template

	dashboardVersion string
)

func init() {
	API7EELicense = os.Getenv("API7_EE_LICENSE")
	if API7EELicense == "" {
		panic("env {API7_EE_LICENSE} is required")
	}

	dashboardVersion = os.Getenv("DASHBOARD_VERSION")
	if dashboardVersion == "" {
		dashboardVersion = "dev"
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
  server:
    persistence:
      enabled: false
postgresql:
{{- if ne .DB "postgres" }}
  builtin: false
{{- end }}
  primary:
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

type responseCreateGateway struct {
	Value    responseCreateGatewayValue `json:"value"`
	ErrorMsg string                     `json:"error_msg"`
}

type responseCreateGatewayValue struct {
	ID             string `json:"id"`
	TokenPlainText string `json:"token_plain_text"`
	Key            string `json:"key"`
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

	chartPathOptions := action.ChartPathOptions{
		RepoURL: "https://charts.api7.ai",
	}

	chartPath, err := chartPathOptions.LocateChart("api7ee3", cli.New())
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "locate helm chart")

	chart, err := loader.Load(chartPath)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "load helm chart")

	install := action.NewInstall(actionConfig)
	install.Namespace = f.kubectlOpts.Namespace
	install.ReleaseName = "api7ee3"

	buf := bytes.NewBuffer(nil)
	_ = valuesTemplate.Execute(buf, map[string]any{
		"DB":  _db,
		"DSN": getDSN(),
		"Tag": dashboardVersion,
	})

	var v map[string]any
	err = yaml.Unmarshal(buf.Bytes(), &v)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "unmarshal values")
	_, err = install.Run(chart, v)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "install dashboard")

	err = f.ensureService("api7ee3-dashboard", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring dashboard service")

	err = f.ensureService("api7-postgresql", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring postgres service")

	err = f.ensureService("api7-prometheus-server", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring prometheus-server service")
}

func (f *Framework) initDashboard() {
	f.deletePods("app.kubernetes.io/name=api7ee3")
	time.Sleep(5 * time.Second)
}

// ParseHTML will parse the doc from login page and generate a map contains id and action.
func (f *Framework) ParseHTML(doc *html.Node) map[string]string {
	var fu func(*html.Node)
	htmlMap := make(map[string]string)
	fu = func(n *html.Node) {
		var (
			name  string
			value string
		)
		for _, attr := range n.Attr {
			if attr.Key == "id" || attr.Key == "name" {
				name = attr.Val
			}
			if attr.Key == "action" || attr.Key == "value" {
				value = attr.Val
			}

			htmlMap[name] = value
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fu(c)
		}
	}
	fu(doc)

	return htmlMap
}

func (f *Framework) GetTokenFromDashboard(gatewayGroupID string) (string, error) {
	respExp := f.DashboardHTTPClient().
		POST("/api/gateway_groups/"+gatewayGroupID+"/instance_token").
		WithHeader("Content-Type", "application/json").
		WithBasicAuth("admin", "admin").
		Expect()

	respExp.Status(200).Body().Contains("token_plain_text")
	body := respExp.Body().Raw()
	// unmarshal into responseCreateGateway
	var response responseCreateGateway
	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		return "", err
	}
	return response.Value.TokenPlainText, nil
}

func (f *Framework) GetDataplaneCertificates(gatewayGroupID string) *v1.DataplaneCertificate {
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
		Value v1.DataplaneCertificate `json:"value"`
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
