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
	"cmp"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/utils"
)

func (t *Translator) TranslateApisixRoute(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute) (result *TranslateResult, err error) {
	result = &TranslateResult{}
	for ruleIndex, http := range ar.Spec.HTTP {
		var timeout *adc.Timeout
		if http.Timeout != nil {
			defaultTimeout := metav1.Duration{Duration: apiv2.DefaultUpstreamTimeout}
			timeout = &adc.Timeout{
				Connect: cmp.Or(int(http.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
				Read:    cmp.Or(int(http.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
				Send:    cmp.Or(int(http.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
			}
		}

		var plugins = make(adc.Plugins)
		// todo: need unit test or e2e test
		for _, plugin := range http.Plugins {
			if !plugin.Enable {
				continue
			}

			config := make(map[string]any)
			if plugin.Config != nil {
				for key, value := range plugin.Config {
					config[key] = json.RawMessage(value.Raw)
				}
			}
			if plugin.SecretRef != "" {
				if secret, ok := tctx.Secrets[types.NamespacedName{Namespace: ar.Namespace, Name: plugin.SecretRef}]; ok {
					for key, value := range secret.Data {
						utils.InsertKeyInMap(key, string(value), config)
					}
				}
			}
			plugins[plugin.Name] = config
		}

		// add Authentication plugins
		if http.Authentication.Enable {
			switch http.Authentication.Type {
			case "keyAuth":
				plugins["key-auth"] = http.Authentication.KeyAuth
			case "basicAuth":
				plugins["basic-auth"] = make(map[string]any)
			case "wolfRBAC":
				plugins["wolf-rbac"] = make(map[string]any)
			case "jwtAuth":
				plugins["jwt-auth"] = http.Authentication.JwtAuth
			case "hmacAuth":
				plugins["hmac-auth"] = make(map[string]any)
			case "ldapAuth":
				plugins["ldap-auth"] = http.Authentication.LDAPAuth
			default:
				plugins["basic-auth"] = make(map[string]any)
			}
		}

		vars, err := http.Match.NginxVars.ToVars()
		if err != nil {
			return nil, err
		}

		var (
			route    = adc.NewDefaultRoute()
			upstream = adc.NewDefaultUpstream()
			service  = adc.NewDefaultService()
			labels   = label.GenLabel(ar)
		)
		// translate to adc.Route
		route.Name = adc.ComposeRouteName(ar.Namespace, ar.Name, http.Name)
		route.ID = id.GenID(route.Name)
		route.Desc = "Created by apisix-ingress-controller, DO NOT modify it manually"
		route.Labels = labels
		route.EnableWebsocket = ptr.To(true)
		route.FilterFunc = http.Match.FilterFunc
		route.Hosts = http.Match.Hosts
		route.Methods = http.Match.Methods
		route.Plugins = plugins
		route.Priority = ptr.To(int64(http.Priority))
		route.RemoteAddrs = http.Match.RemoteAddrs
		route.Timeout = timeout
		route.Uris = http.Match.Paths
		route.Vars = vars

		// translate to adc.Upstream
		var backendErr error
		for _, backend := range http.Backends {
			weight := int32(*cmp.Or(backend.Weight, ptr.To(apiv2.DefaultWeight)))
			backendRef := gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Group:     (*gatewayv1.Group)(&apiv2.GroupVersion.Group),
					Kind:      (*gatewayv1.Kind)(ptr.To("Service")),
					Name:      gatewayv1.ObjectName(backend.ServiceName),
					Namespace: (*gatewayv1.Namespace)(&ar.Namespace),
					Port:      (*gatewayv1.PortNumber)(&backend.ServicePort.IntVal),
				},
				Weight: &weight,
			}
			upNodes, err := t.translateBackendRef(tctx, backendRef)
			if err != nil {
				backendErr = err
				continue
			}
			t.AttachBackendTrafficPolicyToUpstream(backendRef, tctx.BackendTrafficPolicies, upstream)
			upstream.Nodes = append(upstream.Nodes, upNodes...)
		}
		//nolint:staticcheck
		if len(http.Backends) == 0 && len(http.Upstreams) > 0 {
			// FIXME: when the API ApisixUpstream is supported
		}

		// translate to adc.Service
		service.Name = adc.ComposeServiceNameWithRule(ar.Namespace, ar.Name, fmt.Sprintf("%d", ruleIndex))
		service.ID = id.GenID(service.Name)
		service.Labels = ar.Labels
		service.Hosts = http.Match.Hosts
		service.Upstream = upstream
		service.Routes = []*adc.Route{route}

		if backendErr != nil && len(upstream.Nodes) == 0 {
			if service.Plugins == nil {
				service.Plugins = make(map[string]any)
			}
			service.Plugins["fault-injection"] = map[string]any{
				"abort": map[string]any{
					"http_status": 500,
					"body":        "No existing backendRef provided",
				},
			}
		}

		result.Services = append(result.Services, service)
	}

	return result, nil
}
