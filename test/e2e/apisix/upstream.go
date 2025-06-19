// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apisix

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixUpstream", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		err error
	)

	Context("Test ApisixUpstream validation", func() {
		It("validation of externalNodes and discovery", func() {
			const apisixUpstreamSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
spec:
  ingressClassName: apisix
  externalNodes:
  - type: Service
    name: httpbin-service-e2e-test
  discovery:
    serviceName: xx
    type: nacos
`
			const apisixUpstreamSpec1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
spec:
  ingressClassName: apisix
`
			err = s.CreateResourceFromString(apisixUpstreamSpec0)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("has(self.externalNodes)!=has(self.discovery)"))

			err = s.CreateResourceFromString(apisixUpstreamSpec1)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("has(self.externalNodes)!=has(self.discovery)"))

		})
	})
})
