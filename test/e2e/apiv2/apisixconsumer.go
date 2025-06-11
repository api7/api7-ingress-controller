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

package apiv2

import (
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixConsumer", func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)
	Context("Test ApisixConsumer", func() {
		It("Test ApisixConsumer", func() {
			var apisixConsumerSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: defaultapisixconsumer
spec:
  authParameter:
    basicAuth:
      value:
        username: jack
        password: jack-password
`
			applier.MustApplyApisixConsumer(types.NamespacedName{Name: "defaultapisixconsumer", Namespace: s.Namespace()}, apisixConsumerSpec)
		})
	})
})
