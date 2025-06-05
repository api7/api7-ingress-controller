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

package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/apache/apisix-ingress-controller/test/e2e/adminapi"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/apiv2"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/crds"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/gatewayapi"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/ingress"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	f := framework.NewFramework()

	BeforeSuite(f.BeforeSuite)
	AfterSuite(f.AfterSuite)

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting apisix-ingress suite\n")
	RunSpecs(t, "e2e suite")
}
