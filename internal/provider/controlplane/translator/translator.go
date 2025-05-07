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

package translator

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
)

type Translator struct {
	Log logr.Logger
}

type TranslateContext struct {
	BackendRefs      []gatewayv1.BackendRef
	GatewayTLSConfig []gatewayv1.GatewayTLSConfig
	EndpointSlices   map[types.NamespacedName][]discoveryv1.EndpointSlice
	Secrets          map[types.NamespacedName]*corev1.Secret
	PluginConfigs    map[types.NamespacedName]*v1alpha1.PluginConfig
}

type TranslateResult struct {
	Routes   []*v1.Route
	Services []*v1.Service
	SSL      []*v1.Ssl
}

func NewDefaultTranslateContext() *TranslateContext {
	return &TranslateContext{
		EndpointSlices: make(map[types.NamespacedName][]discoveryv1.EndpointSlice),
		Secrets:        make(map[types.NamespacedName]*corev1.Secret),
		PluginConfigs:  make(map[types.NamespacedName]*v1alpha1.PluginConfig),
	}
}
