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
	"time"

	v1 "github.com/api7/api7-ingress-controller/pkg/types/apisix/v1"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
)

// PASSING
var _ = ginkgo.Describe("suite-features: ApisiGlobalRule", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable echo global rule in apisix", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(60 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		var gr *v1.GlobalRule
		for _, g := range grs {
			if _, ok := g.Plugins["echo"]; ok {
				gr = g
			}
		}
		assert.Len(ginkgo.GinkgoT(), gr.Plugins, 1)
		_, ok := gr.Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
		time.Sleep(10 * time.Second)
		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")

		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")
	})

	//Skipping this test because ingress controller doesn't send out the plugin when it is disabled
	//And global rule with 0 plugins is not allowed
	ginkgo.PIt("disable echo global rule in apisix", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		var gr *v1.GlobalRule
		for _, g := range grs {
			if _, ok := g.Plugins["echo"]; ok {
				gr = g
			}
		}
		assert.Len(ginkgo.GinkgoT(), gr.Plugins, 1)
		_, ok := gr.Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")
		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")

		agr = `
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

		grs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		for _, g := range grs {
			if _, ok := g.Plugins["echo"]; ok {
				gr = g
			}
		}
		assert.Len(ginkgo.GinkgoT(), gr.Plugins, 1)
		_, ok = gr.Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, false)

		s.NewAPISIXClient().GET("/anything").Expect().Body().NotContains("hello, world!!")
	})

	ginkgo.It("delete global rule in apisix", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		var gr *v1.GlobalRule
		for _, g := range grs {
			if _, ok := g.Plugins["echo"]; ok {
				gr = g
			}
		}
		assert.Len(ginkgo.GinkgoT(), gr.Plugins, 1)
		_, ok := gr.Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(agr), "deleteing ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 0)

		s.NewAPISIXClient().GET("/anything").Expect().Body().NotContains("hello, world!!")
	})
})
