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
package features

import (
	"fmt"
	"math"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

// PASSING
var _ = ginkgo.Describe("suite-features: traffic split", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("sanity", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			backendSvc2, backendPorts2 := s.HTTPBackend2()
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
       - /get
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   - serviceName: %s
     servicePort: %d
     weight: 5
`, backendSvc, backendPorts[0], backendSvc2, backendPorts2[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			// err := s.EnsureNumApisixUpstreamsCreated(2)
			// assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// Send requests to APISIX.
			var (
				num503 int
				num200 int
			)
			for i := 0; i < 500; i++ {
				// For requests sent to mockbin at /get, 503 will be given.
				// For requests sent to httpbin, 200 will be given.
				resp := s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect()
				status := resp.Raw().StatusCode
				if status != http.StatusOK && status != http.StatusServiceUnavailable {
					assert.FailNow(ginkgo.GinkgoT(), fmt.Sprintf("expected %d or %d but got: %d", http.StatusOK, http.StatusServiceUnavailable, status))
				}
				if status == 200 {
					num200++
					resp.Body().Contains("origin")
				} else {
					num503++
				}
			}
			dev := math.Abs(float64(num200)/float64(num503) - float64(2))
			assert.Less(ginkgo.GinkgoT(), dev, 0.2)
		})

		ginkgo.It("zero-weight", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			backendSvc2, backendPorts2 := s.HTTPBackend2()
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
       - /get
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 100
   - serviceName: %s
     servicePort: %d
     weight: 0
`, backendSvc, backendPorts[0], backendSvc2, backendPorts2[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			time.Sleep(6 * time.Second)
			// err := s.EnsureNumApisixUpstreamsCreated(2)
			// assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// Send requests to APISIX.
			var (
				num503 int
				num200 int
			)
			for i := 0; i < 90; i++ {
				// For requests sent to mockbin at /get, 503 will be given.
				// For requests sent to httpbin, 200 will be given.
				resp := s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect()
				status := resp.Raw().StatusCode
				if status != http.StatusOK && status != http.StatusServiceUnavailable {
					assert.FailNow(ginkgo.GinkgoT(), fmt.Sprintf("expected %d or %d but got: %d", http.StatusOK, http.StatusServiceUnavailable, status))
				}
				if status == 200 {
					num200++
					resp.Body().Contains("origin")
				} else {
					num503++
				}
			}
			assert.Equal(ginkgo.GinkgoT(), num503, 0)
			assert.Equal(ginkgo.GinkgoT(), num200, 90)
		})
	}

	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
