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

var _ = Describe("APISIX Standalone Basic Tests", func() {
	var (
		s *scaffold.APISIXScaffold
	)

	Describe("APISIX HTTP Proxy", func() {
		It("should handle basic HTTP requests", func() {
			httpClient := s.GetHTTPClient()
			Expect(httpClient).NotTo(BeNil())

			// Test basic connectivity
			resp := httpClient.GET("/anything").
				Expect().
				Status(200)

			resp.JSON().Object().ContainsKey("url")
		})
	})
})
