package chore

import (
	"fmt"
	"time"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("suite-chore: apply Apisix Resources and check status", func() {
	PhaseCreateExternalService := func(s *scaffold.Scaffold, name, externalName string) {
		extService := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  type: ExternalName
  externalName: %s
`, name, externalName)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(extService))
	}
	PhaseCreateApisixRoute := func(s *scaffold.Scaffold, name, upstream string) {
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
        - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    upstreams:
    - name: %s
`, name, upstream)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
	}
	PhaseCreateApisixUpstream := func(s *scaffold.Scaffold, name string, nodeType v2.ApisixUpstreamExternalType, nodeName string) {
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  externalNodes:
  - type: %s
    name: %s
`, name, nodeType, nodeName)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(au))
	}

	s := scaffold.NewDefaultV2Scaffold()
	ginkgo.Describe("Apply ApisixRoute, then ApisixUpstream and Service", func() {
		ginkgo.It("should check the status of ApisixRoute and ApisixUpstream with external service", func() {
			PhaseCreateExternalService(s, "httpbin", "httpbin.org")
			PhaseCreateApisixUpstream(s, "httpbin", v2.ExternalTypeService, "httpbin")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin")
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")

			//Check the status of ApisixUpstream resource
			upstreamStatus, err := s.GetApisixResourceStatus("httpbin", "au")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", upstreamStatus.Conditions[0].Message)

			//Check the status of ApisixRoute resource
			routeStatus, err := s.GetApisixResourceStatus("httpbin-route", "ar")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", routeStatus.Conditions[0].Message)
		})
	})

	ginkgo.Describe("Apply ApisixGlobalRule and check status", func() {
		ginkgo.It("should check the status of ApisixGlobalRule", func() {
			agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: false
    config:
      body: "hello, world!!"
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
			time.Sleep(6 * time.Second)
			//Check the status of ApisixGlobalRule resource
			agrStatus, err := s.GetApisixResourceStatus("test-agr-1", "agr")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", agrStatus.Conditions[0].Message)
		})
	})

	ginkgo.Describe("Apply ApisixConsumer and check status", func() {
		ginkgo.It("should check the status of ApisixConsumer", func() {
			ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    jwtAuth:
      value:
        key: foo-key
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "create ApisixConsumer")
			time.Sleep(6 * time.Second)
			//Check the status of ApisixConsumer resource
			acStatus, err := s.GetApisixResourceStatus("foo", "ac")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", acStatus.Conditions[0].Message)
		})
	})

})
