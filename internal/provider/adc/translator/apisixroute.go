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
	"strconv"

	"github.com/api7/gopkg/pkg/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

//nolint:gocyclo
func (t *Translator) TranslateApisixRoute(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute) (result *TranslateResult, err error) {
	result = &TranslateResult{}
	for ruleIndex, rule := range ar.Spec.HTTP {
		var timeout *adc.Timeout
		if rule.Timeout != nil {
			defaultTimeout := metav1.Duration{Duration: apiv2.DefaultUpstreamTimeout}
			timeout = &adc.Timeout{
				Connect: cmp.Or(int(rule.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
				Read:    cmp.Or(int(rule.Timeout.Read.Seconds()), int(defaultTimeout.Seconds())),
				Send:    cmp.Or(int(rule.Timeout.Send.Seconds()), int(defaultTimeout.Seconds())),
			}
		}

		var plugins = make(adc.Plugins)
		for _, plugin := range rule.Plugins {
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
						pkgutils.InsertKeyInMap(key, string(value), config)
					}
				}
			}
			plugins[plugin.Name] = config
		}

		// add Authentication plugins
		if rule.Authentication.Enable {
			switch rule.Authentication.Type {
			case "keyAuth":
				plugins["key-auth"] = rule.Authentication.KeyAuth
			case "basicAuth":
				plugins["basic-auth"] = make(map[string]any)
			case "wolfRBAC":
				plugins["wolf-rbac"] = make(map[string]any)
			case "jwtAuth":
				plugins["jwt-auth"] = rule.Authentication.JwtAuth
			case "hmacAuth":
				plugins["hmac-auth"] = make(map[string]any)
			case "ldapAuth":
				plugins["ldap-auth"] = rule.Authentication.LDAPAuth
			default:
				plugins["basic-auth"] = make(map[string]any)
			}
		}

		vars, err := rule.Match.NginxVars.ToVars()
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
		route.Name = adc.ComposeRouteName(ar.Namespace, ar.Name, rule.Name)
		route.ID = id.GenID(route.Name)
		route.Desc = "Created by apisix-ingress-controller, DO NOT modify it manually"
		route.Labels = labels
		route.EnableWebsocket = ptr.To(true)
		route.FilterFunc = rule.Match.FilterFunc
		route.Hosts = rule.Match.Hosts
		route.Methods = rule.Match.Methods
		route.Plugins = plugins
		route.Priority = ptr.To(int64(rule.Priority))
		route.RemoteAddrs = rule.Match.RemoteAddrs
		route.Timeout = timeout
		route.Uris = rule.Match.Paths
		route.Vars = vars
		for key, value := range ar.GetObjectMeta().GetLabels() {
			route.Labels[key] = value
		}

		//nolint:staticcheck
		if rule.PluginConfigName != "" {
			// FIXME: handle PluginConfig
		}

		// translate backends
		var backendErr error
		for _, backend := range rule.Backends {
			var (
				upNodes adc.UpstreamNodes
			)
			if backend.ResolveGranularity == "service" {
				upNodes, backendErr = t.translateApisixRouteBackendResolveGranularityService(tctx, utils.NamespacedName(ar), backend)
				if backendErr != nil {
					t.Log.Error(backendErr, "failed to translate ApisixRoute backend with ResolveGranularity Service")
					continue
				}
			} else {
				upNodes, backendErr = t.translateApisixRouteBackendResolveGranularityEndpoint(tctx, utils.NamespacedName(ar), backend)
				if backendErr != nil {
					t.Log.Error(backendErr, "failed to translate ApisixRoute backend with ResolveGranularity Endpoint")
					continue
				}
			}

			upstream.Nodes = append(upstream.Nodes, upNodes...)
		}

		var (
			upstreams []*adc.Upstream
		)
		for _, upstreamRef := range rule.Upstreams {
			upsNN := types.NamespacedName{
				Namespace: ar.GetNamespace(),
				Name:      upstreamRef.Name,
			}
			au, ok := tctx.Upstreams[upsNN]
			if !ok {
				log.Debugf("failed to retrieve ApisixUpstream from tctx, ApisixUpstream: %s", upsNN)
				continue
			}
			upstream, err := t.translateApisixUpstream(tctx, au)
			if err != nil {
				t.Log.Error(err, "failed to translate ApisixUpstream", "ApisixUpstream", utils.NamespacedName(au))
				continue
			}

			upstreams = append(upstreams, upstream)
		}

		// If no .http[].backends is used and only .http[].upstreams is used, the first valid upstream is used as service.upstream;
		// Other upstreams are configured in the traffic-split plugin
		if len(rule.Backends) == 0 && len(upstreams) > 0 {
			upstream = upstreams[0]
			upstreams = upstreams[1:]
		}

		var weightedUpstreams []adc.TrafficSplitConfigRuleWeightedUpstream
		for _, item := range upstreams {
			weight, err := strconv.Atoi(item.Labels["meta_weight"])
			if err != nil {
				weight = apiv2.DefaultWeight
			}
			weightedUpstreams = append(weightedUpstreams, adc.TrafficSplitConfigRuleWeightedUpstream{
				Upstream: item,
				Weight:   weight,
			})
		}
		if len(weightedUpstreams) > 0 {
			route.Plugins["traffic-split"] = &adc.TrafficSplitConfig{
				Rules: []adc.TrafficSplitConfigRule{
					{
						WeightedUpstreams: weightedUpstreams,
					},
				},
			}
		}

		// translate to adc.Service
		service.Name = adc.ComposeServiceNameWithRule(ar.Namespace, ar.Name, fmt.Sprintf("%d", ruleIndex))
		service.ID = id.GenID(service.Name)
		service.Labels = label.GenLabel(ar)
		service.Hosts = rule.Match.Hosts
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

func (t *Translator) translateApisixRouteBackendResolveGranularityService(tctx *provider.TranslateContext, arNN types.NamespacedName, backend apiv2.ApisixRouteHTTPBackend) (adc.UpstreamNodes, error) {
	serviceNN := types.NamespacedName{
		Namespace: arNN.Namespace,
		Name:      backend.ServiceName,
	}
	svc, ok := tctx.Services[serviceNN]
	if !ok {
		return nil, errors.Errorf("service not found, ApisixRoute: %s, Service: %s", arNN, serviceNN)
	}
	if svc.Spec.ClusterIP == "" {
		return nil, errors.Errorf("conflict headless service and backend resolve granularity, ApisixRoute: %s, Service: %s", arNN, serviceNN)
	}
	return adc.UpstreamNodes{
		{
			Host:   svc.Spec.ClusterIP,
			Port:   backend.ServicePort.IntValue(),
			Weight: *cmp.Or(backend.Weight, ptr.To(apiv2.DefaultWeight)),
		},
	}, nil
}

func (t *Translator) translateApisixRouteBackendResolveGranularityEndpoint(tctx *provider.TranslateContext, arNN types.NamespacedName, backend apiv2.ApisixRouteHTTPBackend) (adc.UpstreamNodes, error) {
	weight := int32(*cmp.Or(backend.Weight, ptr.To(apiv2.DefaultWeight)))
	backendRef := gatewayv1.BackendRef{
		BackendObjectReference: gatewayv1.BackendObjectReference{
			Group:     (*gatewayv1.Group)(&apiv2.GroupVersion.Group),
			Kind:      (*gatewayv1.Kind)(ptr.To("Service")),
			Name:      gatewayv1.ObjectName(backend.ServiceName),
			Namespace: (*gatewayv1.Namespace)(&arNN.Namespace),
			Port:      (*gatewayv1.PortNumber)(&backend.ServicePort.IntVal),
		},
		Weight: &weight,
	}
	return t.translateBackendRef(tctx, backendRef)
}
