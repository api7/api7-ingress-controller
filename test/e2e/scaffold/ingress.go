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

package scaffold

import (
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

func (s *Scaffold) deployIngress() {
	s.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		AdminKey:       s.AdminKey(),
		AdminTLSVerify: false,
		Namespace:      s.namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint,
		Replicas:       1,
	})
}

func (s *Scaffold) ScaleIngress(replicas int) {
	s.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		AdminKey:       s.AdminKey(),
		AdminTLSVerify: false,
		Namespace:      s.namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint,
		Replicas:       replicas,
	})
}
