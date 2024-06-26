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
package cluster

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

// TODO: FAILING: Because /apisix/prometheus/metrics is not available even after enabling prometheus
var _ = ginkgo.PContext("suite-cluster: ApisixClusterConfig v2", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("enable prometheus", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			assert.Nil(ginkgo.GinkgoT(), s.NewApisixClusterConfig("default", true, true), "creating ApisixClusterConfig")

			defer func() {
				assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixClusterConfig("default", true, true))
			}()

			// Wait until the ApisixClusterConfig create event was delivered.
			time.Sleep(3 * time.Second)

			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  http:
  - name: public-api
    match:
      paths:
      - /apisix/prometheus/metrics
    backends:
    - serviceName: %s
      servicePort: %d
    plugins:
    - name: public-api
      enable: true
`, backendSvc, backendPorts[0])

			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRouteConfig")

			time.Sleep(3 * time.Second)

			grs, err := s.ListApisixGlobalRules()
			assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Equal(ginkgo.GinkgoT(), grs[0].ID, "prometheus")
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			_, ok := grs[0].Plugins["prometheus"]
			assert.Equal(ginkgo.GinkgoT(), ok, true)

			resp := s.NewAPISIXClient().GET("/apisix/prometheus/metrics").Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("# HELP apisix_etcd_modify_indexes Etcd modify index for APISIX keys")
			resp.Body().Contains("# HELP apisix_etcd_reachable Config server etcd reachable from APISIX, 0 is unreachable")
			resp.Body().Contains("# HELP apisix_node_info Info of APISIX node")

			time.Sleep(3 * time.Second)

			resp1 := s.NewAPISIXClient().GET("/apisix/prometheus/metrics").Expect()
			resp1.Status(http.StatusOK)
			resp1.Body().Contains("public-api")
		})
	}

	ginkgo.Describe("suite-cluster: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-cluster: Testing ApisixClusterConfig with IngressClass apisix", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		IngressClass:          "apisix",
	})

	ginkgo.It("ApisiClusterConfig should be ignored", func() {
		// create ApisixClusterConfig resource with ingressClassName: ignore
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: ignore
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 0)
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixClusterConfig resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, "prometheus")
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixClusterConfig resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, "prometheus")
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})

var _ = ginkgo.Describe("suite-cluster: Testing ApisixClusterConfig with IngressClass apisix-and-all", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix-and-all",
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixConsumer resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, "prometheus")
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, "prometheus")
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: watch
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: watch
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, "prometheus")
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})

var _ = ginkgo.Describe("suite-cluster: Enable webhooks to verify IngressClassName", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "webhook",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		EnableWebhooks:        true,
	})

	ginkgo.It("ingressClassName of the ApisixClusterConfig should not be modified", func() {
		apc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: watch
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(apc, ""), "creatint a ApisixClusterConfig")

		apc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: failed
`
		err := s.CreateResourceFromStringWithNamespace(apc, "")
		assert.Error(ginkgo.GinkgoT(), err, "Failed to udpate ApisixClusterConfig")
		assert.Contains(ginkgo.GinkgoT(), err.Error(), "denied the request")
		assert.Contains(ginkgo.GinkgoT(), err.Error(), "The ingressClassName field is not allowed to be modified.")
	})

})
