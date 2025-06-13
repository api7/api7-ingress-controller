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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApisixStatus is the status report for Apisix ingress Resources
type ApisixStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func GetStatus(object client.Object) ApisixStatus {
	switch t := object.(type) {
	case *ApisixConsumer:
		return t.Status
	case *ApisixGlobalRule:
		return t.Status
	case *ApisixPluginConfig:
		return t.Status
	case *ApisixRoute:
		return t.Status
	case *ApisixTls:
		return t.Status
	case *ApisixUpstream:
		return t.Status
	default:
		return ApisixStatus{}
	}
}
