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

package adc

import (
	"fmt"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	types "github.com/apache/apisix-ingress-controller/internal/types"
	networkingv1 "k8s.io/api/networking/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Statuses struct {
	GatewayStatuses   types.Map[k8stypes.NamespacedName, *gatewayv1.GatewayStatus]
	HTTPRouteStatuses types.Map[k8stypes.NamespacedName, *gatewayv1.HTTPRouteStatus]
	IngressStatuses   types.Map[k8stypes.NamespacedName, *networkingv1.IngressStatus]
	ConsumerStatuses  types.Map[k8stypes.NamespacedName, *v1alpha1.ConsumerStatus]
}

func UpdateStatuses(updater status.Updater, statuses *Statuses) {

	for k, v := range statuses.HTTPRouteStatuses.LoadAllOrDelete() {
		updater.Update(status.Update{
			NamespacedName: k,
			Resource:       &gatewayv1.HTTPRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				h, ok := obj.(*gatewayv1.HTTPRoute)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				hCopy := h.DeepCopy()
				hCopy.Status = *v
				return hCopy
			}),
		})
	}

	for k, v := range statuses.GatewayStatuses.LoadAllOrDelete() {
		updater.Update(status.Update{
			NamespacedName: k,
			Resource:       &gatewayv1.Gateway{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				g, ok := obj.(*gatewayv1.Gateway)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				gCopy := g.DeepCopy()
				gCopy.Status = *v
				return gCopy
			}),
		})
	}

	for k, v := range statuses.IngressStatuses.LoadAllOrDelete() {
		updater.Update(status.Update{
			NamespacedName: k,
			Resource:       &networkingv1.Ingress{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				i, ok := obj.(*networkingv1.Ingress)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				iCopy := i.DeepCopy()
				iCopy.Status = *v
				return iCopy
			}),
		})
	}

	for k, v := range statuses.ConsumerStatuses.LoadAllOrDelete() {
		updater.Update(status.Update{
			NamespacedName: k,
			Resource:       &v1alpha1.Consumer{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				c, ok := obj.(*v1alpha1.Consumer)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				cCopy := c.DeepCopy()
				cCopy.Status = *v
				return cCopy
			}),
		})
	}
}
