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

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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

func (t *Translator) TranslateApisixRoute(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute) (result *TranslateResult, err error) {
	result = &TranslateResult{}
	for ruleIndex, rule := range ar.Spec.HTTP {
		service, err := t.translateHTTPRule(tctx, ar, rule, ruleIndex)
		if err != nil {
			return nil, err
		}
		result.Services = append(result.Services, service)
	}
	return result, nil
}

func (t *Translator) translateHTTPRule(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, ruleIndex int) (*adc.Service, error) {
	timeout := t.buildTimeout(rule)
	plugins := t.buildPlugins(tctx, ar, rule)

	vars, err := rule.Match.NginxVars.ToVars()
	if err != nil {
		return nil, err
	}

	route := t.buildRoute(ar, rule, plugins, timeout, vars)
	upstream, backendErr := t.buildUpstream(tctx, ar, rule)
	service := t.buildService(ar, rule, ruleIndex, route, upstream)

	if backendErr != nil && len(upstream.Nodes) == 0 {
		t.addFaultInjectionPlugin(service)
	}

	return service, nil
}

func (t *Translator) buildTimeout(rule apiv2.ApisixRouteHTTP) *adc.Timeout {
	if rule.Timeout == nil {
		return nil
	}
	defaultTimeout := metav1.Duration{Duration: apiv2.DefaultUpstreamTimeout}
	return &adc.Timeout{
		Connect: cmp.Or(int(rule.Timeout.Connect.Seconds()), int(defaultTimeout.Seconds())),
		Read:    cmp.Or(int(rule.Timeout.Read.Seconds()), int(defaultTimeout.Seconds())),
		Send:    cmp.Or(int(rule.Timeout.Send.Seconds()), int(defaultTimeout.Seconds())),
	}
}

func (t *Translator) buildPlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP) adc.Plugins {
	plugins := make(adc.Plugins)

	// Load plugins from referenced PluginConfig
	t.loadPluginConfigPlugins(tctx, ar, rule, plugins)

	// Apply plugins from the route itself
	t.loadRoutePlugins(tctx, ar, rule, plugins)

	// Add authentication plugins
	t.addAuthenticationPlugins(rule, plugins)

	return plugins
}

func (t *Translator) loadPluginConfigPlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	if rule.PluginConfigName == "" {
		return
	}

	pcNamespace := ar.Namespace
	if rule.PluginConfigNamespace != "" {
		pcNamespace = rule.PluginConfigNamespace
	}

	pcKey := types.NamespacedName{Namespace: pcNamespace, Name: rule.PluginConfigName}
	pc, ok := tctx.ApisixPluginConfigs[pcKey]
	if !ok || pc == nil {
		return
	}

	for _, plugin := range pc.Spec.Plugins {
		if !plugin.Enable {
			continue
		}
		config := t.buildPluginConfig(plugin, pc.Namespace, tctx.Secrets)
		plugins[plugin.Name] = config
	}
}

func (t *Translator) loadRoutePlugins(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	for _, plugin := range rule.Plugins {
		if !plugin.Enable {
			continue
		}
		config := t.buildPluginConfig(plugin, ar.Namespace, tctx.Secrets)
		plugins[plugin.Name] = config
	}
}

func (t *Translator) buildPluginConfig(plugin apiv2.ApisixRoutePlugin, namespace string, secrets map[types.NamespacedName]*v1.Secret) map[string]any {
	config := make(map[string]any)
	if plugin.Config != nil {
		for key, value := range plugin.Config {
			config[key] = json.RawMessage(value.Raw)
		}
	}
	if plugin.SecretRef != "" {
		if secret, ok := secrets[types.NamespacedName{Namespace: namespace, Name: plugin.SecretRef}]; ok {
			for key, value := range secret.Data {
				pkgutils.InsertKeyInMap(key, string(value), config)
			}
		}
	}
	return config
}

func (t *Translator) addAuthenticationPlugins(rule apiv2.ApisixRouteHTTP, plugins adc.Plugins) {
	if !rule.Authentication.Enable {
		return
	}

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

func (t *Translator) buildRoute(ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, plugins adc.Plugins, timeout *adc.Timeout, vars adc.Vars) *adc.Route {
	route := adc.NewDefaultRoute()
	route.Name = adc.ComposeRouteName(ar.Namespace, ar.Name, rule.Name)
	route.ID = id.GenID(route.Name)
	route.Desc = "Created by apisix-ingress-controller, DO NOT modify it manually"
	route.Labels = label.GenLabel(ar)
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
	return route
}

func (t *Translator) buildUpstream(tctx *provider.TranslateContext, ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP) (*adc.Upstream, error) {
	upstream := adc.NewDefaultUpstream()
	var backendErr error

	for _, backend := range rule.Backends {
		var upNodes adc.UpstreamNodes
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

	//nolint:staticcheck
	if len(rule.Backends) == 0 && len(rule.Upstreams) > 0 {
		// FIXME: when the API ApisixUpstream is supported
	}

	return upstream, backendErr
}

func (t *Translator) buildService(ar *apiv2.ApisixRoute, rule apiv2.ApisixRouteHTTP, ruleIndex int, route *adc.Route, upstream *adc.Upstream) *adc.Service {
	service := adc.NewDefaultService()
	service.Name = adc.ComposeServiceNameWithRule(ar.Namespace, ar.Name, fmt.Sprintf("%d", ruleIndex))
	service.ID = id.GenID(service.Name)
	service.Labels = label.GenLabel(ar)
	service.Hosts = rule.Match.Hosts
	service.Upstream = upstream
	service.Routes = []*adc.Route{route}
	return service
}

func (t *Translator) addFaultInjectionPlugin(service *adc.Service) {
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
