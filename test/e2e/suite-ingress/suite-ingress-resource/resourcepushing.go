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
package ingress

// import (
// 	"fmt"
// 	"net/http"
// 	"time"

// 	"github.com/api7/api7-ingress-controller/pkg/log"
// 	ginkgo "github.com/onsi/ginkgo/v2"
// 	"github.com/stretchr/testify/assert"

// 	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
// )

// var _ = ginkgo.Describe("suite-ingress-resource: ApisixRoute Testing", func() {
// 	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
// 		s := scaffoldFunc()
// 		ginkgo.It("create and then scale upstream pods to 2 ", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			err := s.EnsureNumApisixRoutesCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			err = s.EnsureNumApisixUpstreamsCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
// 			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instances")
// 			assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPodsAvailable(), "waiting for all httpbin pods ready")

// 			ups, err := s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
// 			assert.Len(ginkgo.GinkgoT(), ups[0].Upstream.Nodes, 2, "upstreams nodes not expect")
// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
// 		})

// 		ginkgo.It("create, update, then remove", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			err := s.EnsureNumApisixRoutesCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			err = s.EnsureNumApisixUpstreamsCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
// 			err = s.EnsureNumApisixPluginConfigCreated(0)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

// 			// update
// 			apisixRoute = fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//       exprs:
//       - subject:
//           scope: Header
//           name: X-Foo
//         op: Equal
//         value: "barbaz"
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			// EnsureNumApisixRoutesCreated cannot be used to ensure update Correctness.
// 			time.Sleep(6 * time.Second)

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbaz").Expect().Status(http.StatusOK)

// 			// remove
// 			assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))

// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")

// 			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound).Body().Raw()
// 			assert.Contains(ginkgo.GinkgoT(), body, "404 Route Not Found")
// 		})

// 		ginkgo.It("create, update, remove k8s service, remove ApisixRoute", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			err := s.EnsureNumApisixRoutesCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			err = s.EnsureNumApisixUpstreamsCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
// 			err = s.EnsureNumApisixPluginConfigCreated(0)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

// 			// update
// 			apisixRoute = fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//       exprs:
//       - subject:
//           scope: Header
//           name: X-Foo
//         op: Equal
//         value: "barbaz"
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			// EnsureNumApisixRoutesCreated cannot be used to ensure update Correctness.
// 			time.Sleep(6 * time.Second)

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbaz").Expect().Status(http.StatusOK)
// 			// remove k8s service first
// 			// assert.Nil(ginkgo.GinkgoT(), s.DeleteHTTPBINService())
// 			// remove
// 			assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")

// 			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound).Body().Raw()
// 			assert.Contains(ginkgo.GinkgoT(), body, "404 Route Not Found")
// 		})

// 		ginkgo.It("change route rule name", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0))

// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing routes in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1)

// 			upstreams, err := s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), upstreams, 1)

// 			pluginConfigs, err := s.ListApisixPluginConfig()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing pluginConfigs in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), pluginConfigs, 0)

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

// 			apisixRoute = fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1_1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /headers
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0))

// 			newRoutes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing routes in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), newRoutes, 1)
// 			newUpstreams, err := s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), newUpstreams, 1)
// 			newPluginConfigs, err := s.ListApisixPluginConfig()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing pluginConfigs in APISIX")
// 			assert.Len(ginkgo.GinkgoT(), newPluginConfigs, 0)

// 			// Upstream doesn't change.
// 			assert.Equal(ginkgo.GinkgoT(), newUpstreams[0].ID, upstreams[0].ID)
// 			assert.Equal(ginkgo.GinkgoT(), newUpstreams[0].Name, upstreams[0].Name)

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().
// 				Status(http.StatusNotFound).
// 				Body().Contains("404 Route Not Found")

// 			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().
// 				Status(http.StatusOK)
// 		})

// 		ginkgo.It("same route rule name between two ApisixRoute objects", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// ---
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route-2
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /headers
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0], backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0))

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().
// 				Status(http.StatusOK).
// 				Body().
// 				Contains("origin")
// 			s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().
// 				Status(http.StatusOK).
// 				Body().
// 				Contains("headers").
// 				Contains("httpbin.com")
// 		})

// 		ginkgo.It("route priority", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     priority: 1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
//   - name: rule2
//     priority: 2
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//       exprs:
//       - subject:
//           scope: Header
//           name: X-Foo
//         op: Equal
//         value: barbazbar
//     backends:
//     - serviceName: %s
//       servicePort: %d
//     plugins:
//     - name: request-id
//       enable: true
// `, backendSvc, backendSvcPort[0], backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0))

// 			// Hit rule1
// 			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusOK)
// 			resp.Body().Contains("origin")
// 			resp.Header("X-Request-Id").Empty()

// 			// Hit rule2
// 			resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbazbar").Expect()
// 			resp.Status(http.StatusOK)
// 			resp.Body().Contains("origin")
// 			resp.Header("X-Request-Id").NotEmpty()
// 		})

// 		ginkgo.It("verify route/upstream/pluginConfig items", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     priority: 1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0))

// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1)
// 			name := s.Namespace() + "_" + "httpbin-route" + "_" + "rule1"
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Name, name)
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Paths, []string{"/ip"})
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Hosts, []string{"httpbin.com"})
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Desc,
// 				"Created by apisix-ingress-controller, DO NOT modify it manually")
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Labels["managed-by"], "apisix-ingress-controller")

// 			ups, err := s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
// 			assert.Len(ginkgo.GinkgoT(), ups, 1)
// 			assert.Equal(ginkgo.GinkgoT(), ups[0].Desc,
// 				"Created by apisix-ingress-controller, DO NOT modify it manually")
// 			assert.Equal(ginkgo.GinkgoT(), ups[0].Labels["managed-by"], "apisix-ingress-controller")

// 			pluginConfigs, err := s.ListApisixPluginConfig()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing pluginConfigs")
// 			assert.Len(ginkgo.GinkgoT(), pluginConfigs, 0)

// 			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusOK)
// 			resp.Body().Contains("origin")

// 			resp = s.NewAPISIXClient().GET("/ip").Expect()
// 			resp.Status(http.StatusNotFound)
// 			resp.Body().Contains("404 Route Not Found")
// 		})

// 		ginkgo.It("service is referenced by two ApisixRoutes", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			ar1 := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route-1
// spec:
//   http:
//   - name: rule1
//     priority: 1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])
// 			ar2 := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route-2
// spec:
//   http:
//   - name: rule1
//     priority: 1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /status/200
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar1))
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar2))

// 			err := s.EnsureNumApisixRoutesCreated(2)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
// 			assert.Len(ginkgo.GinkgoT(), routes, 2)

// 			ups, err := s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
// 			assert.Len(ginkgo.GinkgoT(), ups, 1)
// 			assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[0].ServiceID)
// 			assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[1].ServiceID)

// 			pluginConfigs, err := s.ListApisixPluginConfig()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing pluginConfigs")
// 			assert.Len(ginkgo.GinkgoT(), pluginConfigs, 0)

// 			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusOK)
// 			resp.Body().Contains("origin")

// 			resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusOK)

// 			// Delete ar1
// 			err = s.RemoveResourceByString(ar1)
// 			assert.Nil(ginkgo.GinkgoT(), err)

// 			err = s.EnsureNumApisixRoutesCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			routes, err = s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1)
// 			name := s.Namespace() + "_" + "httpbin-route-2" + "_" + "rule1"
// 			assert.Equal(ginkgo.GinkgoT(), routes[0].Name, name)

// 			// As httpbin service is referenced by ar2, the corresponding upstream still exists.
// 			ups, err = s.ListApisixServices()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
// 			assert.Len(ginkgo.GinkgoT(), ups, 1)
// 			assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[0].ServiceID)

// 			// As httpbin service is referenced by ar2, the corresponding PluginConfig still doesn't exist.
// 			pluginConfigs, err = s.ListApisixPluginConfig()
// 			assert.Nil(ginkgo.GinkgoT(), err, "listing pluginConfigs")
// 			assert.Len(ginkgo.GinkgoT(), pluginConfigs, 0)

// 			resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusNotFound)
// 			resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusOK)

// 			// Delete ar2
// 			assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(ar2))

// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "Checking number of upstreams")
// 			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(0), "Checking number of upstreams")

// 			resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
// 			resp.Status(http.StatusNotFound)
// 		})

// 		ginkgo.It("k8s service is created later than ApisixRoute", func() {
// 			createSvc := func() {
// 				_httpbinDeploymentTemplate := `
// apiVersion: apps/v1
// kind: Deployment
// metadata:
//   name: httpbin-temp
// spec:
//   replicas: 1
//   selector:
//     matchLabels:
//       app: httpbin-temp
//   strategy:
//     rollingUpdate:
//       maxSurge: 50%
//       maxUnavailable: 1
//     type: RollingUpdate
//   template:
//     metadata:
//       labels:
//         app: httpbin-temp
//     spec:
//       terminationGracePeriodSeconds: 0
//       containers:
//         - livenessProbe:
//             failureThreshold: 3
//             initialDelaySeconds: 2
//             periodSeconds: 5
//             successThreshold: 1
//             tcpSocket:
//               port: 80
//             timeoutSeconds: 2
//           readinessProbe:
//             failureThreshold: 3
//             initialDelaySeconds: 2
//             periodSeconds: 5
//             successThreshold: 1
//             tcpSocket:
//               port: 80
//             timeoutSeconds: 2
//           image: "127.0.0.1:5000/httpbin:dev"
//           imagePullPolicy: IfNotPresent
//           name: httpbin-temp
//           ports:
//             - containerPort: 80
//               name: "http"
//               protocol: "TCP"
// `
// 				_httpService := `
// apiVersion: v1
// kind: Service
// metadata:
//   name: httpbin-temp
// spec:
//   selector:
//     app: httpbin-temp
//   ports:
//     - name: http
//       port: 80
//       protocol: TCP
//       targetPort: 80
//   type: ClusterIP
// `

// 				err := s.CreateResourceFromString(s.FormatRegistry(_httpbinDeploymentTemplate))
// 				if err != nil {
// 					log.Errorf(err.Error())
// 				}
// 				assert.Nil(ginkgo.GinkgoT(), err, "create temp httpbin deployment")
// 				assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_httpService), "create temp httpbin service")
// 			}

// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - httpbin.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, "httpbin-temp", 80)

// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute), "creating ApisixRoute")
// 			// We don't have service yet, so route/upstream==0
// 			err := s.EnsureNumApisixRoutesCreated(0)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			err = s.EnsureNumApisixUpstreamsCreated(0)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

// 			createSvc()
// 			time.Sleep(time.Second * 10)
// 			err = s.EnsureNumApisixRoutesCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
// 			err = s.EnsureNumApisixUpstreamsCreated(1)
// 			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

// 			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)
// 		})
// 	}

// 	ginkgo.Describe("suite-ingress-resource: scaffold v2", func() {
// 		suites(scaffold.NewDefaultV2Scaffold)
// 	})
// })
