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
package plugins

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-security: uri-blocker plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("sanity", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
       - /status/200
       - /headers
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: uri-blocker
     enable: true
     config:
       rejected_code: 403
       block_rules:
       - /status/200
       - /headers
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			//Check the status of ApisixRoute resource
			routeStatus, err := s.GetApisixResourceStatus("httpbin-route", "ar")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", routeStatus.Conditions[0].Message)
			s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.org").
				Expect().
				Status(403)
			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").
				Expect().
				Status(403)
			s.NewAPISIXClient().GET("/status/206").WithHeader("Host", "httpbin.org").
				Expect().
				Status(404).
				Body().
				Contains("404 Route Not Found")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
		})

		ginkgo.It("disable plugin", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
       - /status/200
       - /headers
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: uri-blocker
     enable: false
     config:
       rejected_code: 403
       block_rules:
       - /status/200
       - /headers
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			//Check the status of ApisixRoute resource
			routeStatus, err := s.GetApisixResourceStatus("httpbin-route", "ar")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", routeStatus.Conditions[0].Message)
			s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200)
			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("httpbin.org")
			s.NewAPISIXClient().GET("/status/206").WithHeader("Host", "httpbin.org").
				Expect().
				Status(404).
				Body().
				Contains("404 Route Not Found")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
		})

		ginkgo.It("enable plugin and then delete it", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
       - /status/200
       - /headers
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: uri-blocker
     enable: true
     config:
       rejected_code: 403
       block_rules:
       - /status/200
       - /headers
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(2 * time.Second)
			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			//Check the status of ApisixRoute resource
			routeStatus, err := s.GetApisixResourceStatus("httpbin-route", "ar")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", routeStatus.Conditions[0].Message)
			s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusForbidden)
			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusForbidden)
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")

			ar = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
       - /status/200
       - /headers
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			// EnsureNumApisixRoutesCreated cannot be used to ensure update Correctness.
			time.Sleep(6 * time.Second)
			//Check the status of ApisixRoute resource
			routeStatus, err = s.GetApisixResourceStatus("httpbin-route", "ar")
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), "Sync Successfully", routeStatus.Conditions[0].Message)
			s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK)
			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Contains("httpbin.org")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
		})
	}

	ginkgo.Describe("suite-plugins-security: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
