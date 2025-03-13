// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adminapi

import (
	"context"
	"time"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

var _ = PDescribe("Test Dashboard admin-api sdk", func() {
	s := scaffold.NewDefaultScaffold()

	Context("Service and Route", func() {
		var (
			serviceDefault = &v1.Service{
				Metadata: v1.Metadata{
					ID:   "test-httpbin-id",
					Name: "test-httpbin",
				},
				Upstream: &v1.Upstream{
					Type: "roundrobin",
					Nodes: []v1.UpstreamNode{
						{
							Host:   "httpbin-service-e2e-test",
							Port:   80,
							Weight: 1,
						},
					},
				},
			}
			routeDefault = &v1.Route{
				Metadata: v1.Metadata{
					Name: "test-route",
					ID:   "test-routes",
				},
				ServiceID: "test-httpbin-id",
				Paths:     []string{"/headers", "/ip", "/get"},
			}

			routeAnything = &v1.Route{
				Metadata: v1.Metadata{
					Name: "test-route",
					ID:   "test-route-id",
				},
				ServiceID: "test-httpbin-id",
				Paths:     []string{"/anything"},
			}
		)

		It("Test service and route", func() {
			By("create service and route")
			_, err := s.DefaultDataplaneResource().Service().Create(context.Background(), serviceDefault)
			Expect(err).ToNot(HaveOccurred())

			_, err = s.DefaultDataplaneResource().Route().Create(context.Background(), routeDefault)
			Expect(err).ToNot(HaveOccurred())

			// TODO: use control-api to check the service and route
			time.Sleep(6 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				WithHeader("Test-Header", "t1").
				Expect().
				Status(200).
				Body().
				Contains("Test-Header")

			By("enable plugin in route")
			route2 := routeDefault.DeepCopy()
			route2.Plugins = v1.Plugins{
				"proxy-rewrite": map[string]any{
					"headers": map[string]any{
						"add": map[string]any{
							"X-Header-1": "v1",
						},
					},
				},
			}
			_, err = s.DefaultDataplaneResource().Route().Update(context.Background(), route2)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(6 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				Expect().
				Status(200).
				Body().
				Contains("X-Header-1")

			By("create another route")
			_, err = s.DefaultDataplaneResource().Route().Create(context.Background(), routeAnything)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(6 * time.Second)

			s.NewAPISIXClient().
				GET("/anything").
				Expect().
				Status(200)

			By("enable plugin in service")
			service2 := serviceDefault.DeepCopy()
			service2.Plugins = v1.Plugins{
				"proxy-rewrite": map[string]any{
					"headers": map[string]any{
						"add": map[string]any{
							"X-Header-2": "v2",
						},
					},
				},
			}

			_, err = s.DefaultDataplaneResource().Service().Update(context.Background(), service2)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(6 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				Expect().
				Status(200).
				Body().
				Contains("X-Header-1").
				NotContains("X-Header-2")

			s.NewAPISIXClient().
				GET("/anything").
				Expect().
				Status(200).
				Body().
				Contains("X-Header-2").
				NotContains("X-Header-1")

			By("delete service and route")

			err = s.DefaultDataplaneResource().Route().Delete(context.Background(), routeDefault)
			Expect(err).ToNot(HaveOccurred())
			err = s.DefaultDataplaneResource().Route().Delete(context.Background(), routeAnything)
			Expect(err).ToNot(HaveOccurred())
			err = s.DefaultDataplaneResource().Service().Delete(context.Background(), serviceDefault)
			Expect(err).ToNot(HaveOccurred())

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(routes).To(BeEmpty())

			services, err := s.DefaultDataplaneResource().Service().List(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(services).To(BeEmpty())
		})

		It("Test apply config with https", func() {
			By("create service and route")
			_, err := s.DefaultDataplaneResourceHTTPS().Service().Create(context.Background(), serviceDefault)
			Expect(err).ToNot(HaveOccurred())

			_, err = s.DefaultDataplaneResourceHTTPS().Route().Create(context.Background(), routeDefault)
			Expect(err).ToNot(HaveOccurred())

			// TODO: use control-api to check the service and route
			time.Sleep(6 * time.Second)

			s.NewAPISIXClient().
				GET("/headers").
				WithHeader("Test-Header", "t1").
				Expect().
				Status(200).
				Body().
				Contains("Test-Header")
		})
	})

	Context("Test Plugin metadata", func() {
		It("Update plugin meatadata", func() {
			// update plugin metadata
			datadog := &v1.PluginMetadata{
				Name: "datadog",
				Metadata: map[string]any{
					"host": "172.168.45.29",
					"port": float64(8126),
					"constant_tags": []any{
						"source:apisix",
						"service:custom",
					},
					"namespace": "apisix",
				},
			}

			_, err := s.DefaultDataplaneResource().PluginMetadata().Update(context.Background(), datadog)
			Expect(err).ToNot(HaveOccurred())

			// TODO: use control-api to check the plugin metadata
			time.Sleep(6 * time.Second)

			// update plugin metadata
			updated := &v1.PluginMetadata{
				Name: "datadog",
				Metadata: map[string]any{
					"host": "127.0.0.1",
					"port": float64(8126),
					"constant_tags": []any{
						"source:ingress",
						"service:custom",
					},
					"namespace": "ingress",
				},
			}
			_, err = s.DefaultDataplaneResource().PluginMetadata().Update(context.Background(), updated)
			Expect(err).ToNot(HaveOccurred())

			// TODO: use control-api to check the plugin metadata
		})
	})
})
