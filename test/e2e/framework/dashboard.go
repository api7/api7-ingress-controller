package framework

import (
	"bytes"
	_ "embed"
	"os"
	"text/template"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/onsi/gomega"
	"golang.org/x/net/html"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	valuesTemplate *template.Template
)

func init() {
	tmpl, err := template.New("values.yaml").Parse(`
dashboard:
  image:
    repository: hkccr.ccs.tencentyun.com/api7-dev/api7-ee-3-integrated
    pullPolicy: IfNotPresent
    tag: v3.2.14.2
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
    tag: v3.2.14.2
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
  fallback_cp:
    aws_s3:
      access_key: "some_access_key"
      secret_key: "some_secret"
      region: "api7-test"
      resource_bucket: "to-push-resource-data"
      config_bucket: "to-push-config-data"
      custom_endpoint: "http://{{ .S3Endpoint }}"
    cron_spec: "@every 1s"
  plugins:
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
`)
	if err != nil {
		panic(err)
	}
	valuesTemplate = tmpl
}

func (f *Framework) deploy() {
	debug := func(format string, v ...any) {
		log.Infof(format, v...)
	}

	kubeConfigPath := os.Getenv("KUBECONFIG")
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(kube.GetConfig(kubeConfigPath, "", f.kubectlOpts.Namespace), f.kubectlOpts.Namespace, "memory", debug)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "init helm action config")

	chartPathOptions := action.ChartPathOptions{
		RepoURL: "https://charts.api7.ai",
	}

	chartPath, err := chartPathOptions.LocateChart("api7ee3", cli.New())
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "locate helm chart")

	chart, err := loader.Load(chartPath)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "load helm chart")

	install := action.NewInstall(actionConfig)
	install.Namespace = f.kubectlOpts.Namespace
	install.ReleaseName = "api7ee3"

	buf := bytes.NewBuffer(nil)
	_ = valuesTemplate.Execute(buf, map[string]any{
		"DB":         _db,
		"DSN":        getDSN(),
		"S3Endpoint": f.getMockS3ServiceIP(),
	})

	var v map[string]any
	err = yaml.Unmarshal(buf.Bytes(), &v)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "unmarshal values")
	_, err = install.Run(chart, v)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "install dashboard")

	err = f.ensureService("api7ee3-dashboard", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring dashboard service")

	if _db == postgres {
		err = f.ensureService("api7-postgresql", _namespace, 1)
		f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring postgres service")
	}

	err = f.ensureService("api7-prometheus-server", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring prometheus-server service")
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
