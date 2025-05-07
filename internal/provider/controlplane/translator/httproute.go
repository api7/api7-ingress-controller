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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/gopkg/pkg/log"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/internal/controller/label"
	"github.com/api7/api7-ingress-controller/internal/id"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

func (t *Translator) fillPluginsFromHTTPRouteFilters(
	plugins v1.Plugins,
	namespace string,
	filters []gatewayv1.HTTPRouteFilter,
	matches []gatewayv1.HTTPRouteMatch,
	tctx *provider.TranslateContext,
) {
	for _, filter := range filters {
		switch filter.Type {
		case gatewayv1.HTTPRouteFilterRequestHeaderModifier:
			t.fillPluginFromHTTPRequestHeaderFilter(plugins, filter.RequestHeaderModifier)
		case gatewayv1.HTTPRouteFilterRequestRedirect:
			t.fillPluginFromHTTPRequestRedirectFilter(plugins, filter.RequestRedirect)
		case gatewayv1.HTTPRouteFilterRequestMirror:
			t.fillPluginFromHTTPRequestMirrorFilter(plugins, namespace, filter.RequestMirror)
		case gatewayv1.HTTPRouteFilterURLRewrite:
			t.fillPluginFromURLRewriteFilter(plugins, filter.URLRewrite, matches)
		case gatewayv1.HTTPRouteFilterResponseHeaderModifier:
			t.fillPluginFromHTTPResponseHeaderFilter(plugins, filter.ResponseHeaderModifier)
		case gatewayv1.HTTPRouteFilterExtensionRef:
			t.fillPluginFromExtensionRef(plugins, namespace, filter.ExtensionRef, tctx)
		}
	}
}

func (t *Translator) fillPluginFromExtensionRef(plugins v1.Plugins, namespace string, extensionRef *gatewayv1.LocalObjectReference, tctx *provider.TranslateContext) {
	if extensionRef == nil {
		return
	}
	if extensionRef.Kind == "PluginConfig" {
		pluginconfig := tctx.PluginConfigs[types.NamespacedName{
			Namespace: namespace,
			Name:      string(extensionRef.Name),
		}]
		for _, plugin := range pluginconfig.Spec.Plugins {
			pluginName := plugin.Name
			plugins[pluginName] = plugin.Config
			log.Errorw("plugin config", zap.String("namespace", namespace), zap.Any("plugin_config", plugin))
		}
		log.Errorw("plugin config", zap.String("namespace", namespace), zap.Any("plugins", plugins))
	}
}

func (t *Translator) fillPluginFromURLRewriteFilter(plugins v1.Plugins, urlRewrite *gatewayv1.HTTPURLRewriteFilter, matches []gatewayv1.HTTPRouteMatch) {
	pluginName := v1.PluginProxyRewrite
	obj := plugins[pluginName]
	var plugin *v1.RewriteConfig
	if obj == nil {
		plugin = &v1.RewriteConfig{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*v1.RewriteConfig)
	}
	if urlRewrite.Hostname != nil {
		plugin.Host = string(*urlRewrite.Hostname)
	}

	if urlRewrite.Path != nil {
		switch urlRewrite.Path.Type {
		case gatewayv1.FullPathHTTPPathModifier:
			plugin.RewriteTarget = *urlRewrite.Path.ReplaceFullPath
		case gatewayv1.PrefixMatchHTTPPathModifier:
			prefixPaths := make([]string, 0, len(matches))
			for _, match := range matches {
				if match.Path == nil || match.Path.Type == nil || *match.Path.Type != gatewayv1.PathMatchPathPrefix {
					continue
				}
				prefixPaths = append(prefixPaths, *match.Path.Value)
			}
			regexPattern := "^(" + strings.Join(prefixPaths, "|") + ")" + "/(.*)"
			replaceTarget := *urlRewrite.Path.ReplacePrefixMatch
			regexTarget := replaceTarget + "/$2"

			plugin.RewriteTargetRegex = []string{
				regexPattern,
				regexTarget,
			}
		}
	}
}

func (t *Translator) fillPluginFromHTTPRequestHeaderFilter(plugins v1.Plugins, reqHeaderModifier *gatewayv1.HTTPHeaderFilter) {
	pluginName := v1.PluginProxyRewrite
	obj := plugins[pluginName]
	var plugin *v1.RewriteConfig
	if obj == nil {
		plugin = &v1.RewriteConfig{
			Headers: &v1.Headers{
				Add:    make(map[string]string, len(reqHeaderModifier.Add)),
				Set:    make(map[string]string, len(reqHeaderModifier.Set)),
				Remove: make([]string, 0, len(reqHeaderModifier.Remove)),
			},
		}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*v1.RewriteConfig)
	}
	for _, header := range reqHeaderModifier.Add {
		val := plugin.Headers.Add[string(header.Name)]
		if val != "" {
			val += ", " + header.Value
		} else {
			val = header.Value
		}
		plugin.Headers.Add[string(header.Name)] = val
	}
	for _, header := range reqHeaderModifier.Set {
		plugin.Headers.Set[string(header.Name)] = header.Value
	}
	plugin.Headers.Remove = append(plugin.Headers.Remove, reqHeaderModifier.Remove...)
}

func (t *Translator) fillPluginFromHTTPResponseHeaderFilter(plugins v1.Plugins, respHeaderModifier *gatewayv1.HTTPHeaderFilter) {
	pluginName := v1.PluginResponseRewrite
	obj := plugins[pluginName]
	var plugin *v1.ResponseRewriteConfig
	if obj == nil {
		plugin = &v1.ResponseRewriteConfig{
			Headers: &v1.ResponseHeaders{
				Add:    make([]string, 0, len(respHeaderModifier.Add)),
				Set:    make(map[string]string, len(respHeaderModifier.Set)),
				Remove: make([]string, 0, len(respHeaderModifier.Remove)),
			},
		}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*v1.ResponseRewriteConfig)
	}
	for _, header := range respHeaderModifier.Add {
		plugin.Headers.Add = append(plugin.Headers.Add, fmt.Sprintf("%s: %s", header.Name, header.Value))
	}
	for _, header := range respHeaderModifier.Set {
		plugin.Headers.Set[string(header.Name)] = header.Value
	}
	plugin.Headers.Remove = append(plugin.Headers.Remove, respHeaderModifier.Remove...)
}

func (t *Translator) fillPluginFromHTTPRequestMirrorFilter(plugins v1.Plugins, namespace string, reqMirror *gatewayv1.HTTPRequestMirrorFilter) {
	pluginName := v1.PluginProxyMirror
	obj := plugins[pluginName]

	var plugin *v1.RequestMirror
	if obj == nil {
		plugin = &v1.RequestMirror{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*v1.RequestMirror)
	}

	var (
		port = 80
		ns   = namespace
	)
	if reqMirror.BackendRef.Port != nil {
		port = int(*reqMirror.BackendRef.Port)
	}
	if reqMirror.BackendRef.Namespace != nil {
		ns = string(*reqMirror.BackendRef.Namespace)
	}

	host := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", reqMirror.BackendRef.Name, ns, port)

	plugin.Host = host
}

func (t *Translator) fillPluginFromHTTPRequestRedirectFilter(plugins v1.Plugins, reqRedirect *gatewayv1.HTTPRequestRedirectFilter) {
	pluginName := v1.PluginRedirect
	obj := plugins[pluginName]

	var plugin *v1.RedirectConfig
	if obj == nil {
		plugin = &v1.RedirectConfig{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*v1.RedirectConfig)
	}
	var uri string

	code := 302
	if reqRedirect.StatusCode != nil {
		code = *reqRedirect.StatusCode
	}

	hostname := "$host"
	if reqRedirect.Hostname != nil {
		hostname = string(*reqRedirect.Hostname)
	}

	scheme := "$scheme"
	if reqRedirect.Scheme != nil {
		scheme = *reqRedirect.Scheme
	}

	if reqRedirect.Port != nil {
		uri = fmt.Sprintf("%s://%s:%d$request_uri", scheme, hostname, int(*reqRedirect.Port))
	} else {
		uri = fmt.Sprintf("%s://%s$request_uri", scheme, hostname)
	}
	plugin.RetCode = code
	plugin.URI = uri
}

func (t *Translator) translateEndpointSlice(endpointSlices []discoveryv1.EndpointSlice) v1.UpstreamNodes {
	var nodes v1.UpstreamNodes
	if len(endpointSlices) == 0 {
		return nodes
	}
	for _, endpointSlice := range endpointSlices {
		for _, port := range endpointSlice.Ports {
			for _, endpoint := range endpointSlice.Endpoints {
				for _, addr := range endpoint.Addresses {
					node := v1.UpstreamNode{
						Host:   addr,
						Port:   int(*port.Port),
						Weight: 1,
					}
					nodes = append(nodes, node)
				}
			}
		}
	}

	return nodes
}

func (t *Translator) translateBackendRef(tctx *provider.TranslateContext, ref gatewayv1.BackendRef) *v1.Upstream {
	upstream := v1.NewDefaultUpstream()
	endpointSlices := tctx.EndpointSlices[types.NamespacedName{
		Namespace: string(*ref.Namespace),
		Name:      string(ref.Name),
	}]

	upstream.Nodes = t.translateEndpointSlice(endpointSlices)
	return upstream
}

func (t *Translator) TranslateHTTPRoute(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) (*TranslateResult, error) {
	result := &TranslateResult{}

	hosts := make([]string, 0, len(httpRoute.Spec.Hostnames))
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	rules := httpRoute.Spec.Rules

	for i, rule := range rules {

		var weightedUpstreams []v1.TrafficSplitConfigRuleWeightedUpstream
		upstreams := []*v1.Upstream{}
		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(httpRoute.Namespace)
				backend.Namespace = &namespace
			}
			upstream := t.translateBackendRef(tctx, backend.BackendRef)
			upstream.Labels["name"] = string(backend.Name)
			upstream.Labels["namespace"] = string(*backend.Namespace)
			upstreams = append(upstreams, upstream)
			if len(upstream.Nodes) == 0 {
				upstream.Nodes = v1.UpstreamNodes{
					{
						Host:   "0.0.0.0",
						Port:   80,
						Weight: 100,
					},
				}
			}

			weight := 100
			if backend.Weight != nil {
				weight = int(*backend.Weight)
			}
			weightedUpstreams = append(weightedUpstreams, v1.TrafficSplitConfigRuleWeightedUpstream{
				Upstream: upstream,
				Weight:   weight,
			})
		}

		if len(upstreams) == 0 {
			upstream := v1.NewDefaultUpstream()
			upstream.Nodes = v1.UpstreamNodes{
				{
					Host:   "0.0.0.0",
					Port:   80,
					Weight: 100,
				},
			}
			upstreams = append(upstreams, upstream)
		}

		service := v1.NewDefaultService()
		service.Upstream = upstreams[0]
		if len(weightedUpstreams) > 1 {
			weightedUpstreams[0].Upstream = nil
			service.Plugins["traffic-split"] = &v1.TrafficSplitConfig{
				Rules: []v1.TrafficSplitConfigRule{
					{
						WeightedUpstreams: weightedUpstreams,
					},
				},
			}
		}

		service.Name = v1.ComposeServiceNameWithRule(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d", i))
		service.ID = id.GenID(service.Name)
		service.Labels = label.GenLabel(httpRoute)
		service.Hosts = hosts
		t.fillPluginsFromHTTPRouteFilters(service.Plugins, httpRoute.GetNamespace(), rule.Filters, rule.Matches, tctx)

		result.Services = append(result.Services, service)

		matches := rule.Matches
		if len(matches) == 0 {
			defaultType := gatewayv1.PathMatchPathPrefix
			defaultValue := "/"
			matches = []gatewayv1.HTTPRouteMatch{
				{
					Path: &gatewayv1.HTTPPathMatch{
						Type:  &defaultType,
						Value: &defaultValue,
					},
				},
			}
		}

		for j, match := range matches {
			route, err := t.translateGatewayHTTPRouteMatch(&match)
			if err != nil {
				return nil, err
			}

			name := v1.ComposeRouteName(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			route.Name = name
			route.ID = id.GenID(name)
			route.Labels = label.GenLabel(httpRoute)
			route.ServiceID = service.ID
			result.Routes = append(result.Routes, route)
		}
	}

	return result, nil
}

// NOTE: Dashboard not support Vars, matches only support Path, not support Headers, QueryParams
func (t *Translator) translateGatewayHTTPRouteMatch(match *gatewayv1.HTTPRouteMatch) (*v1.Route, error) {
	route := v1.NewDefaultRoute()

	if match.Path != nil {
		switch *match.Path.Type {
		case gatewayv1.PathMatchExact:
			route.Paths = []string{*match.Path.Value}
		case gatewayv1.PathMatchPathPrefix:
			route.Paths = []string{*match.Path.Value + "*"}
		case gatewayv1.PathMatchRegularExpression:
			var this []v1.StringOrSlice
			this = append(this, v1.StringOrSlice{
				StrVal: "uri",
			})
			this = append(this, v1.StringOrSlice{
				StrVal: "~~",
			})
			this = append(this, v1.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		default:
			return nil, errors.New("unknown path match type " + string(*match.Path.Type))
		}
	}

	if len(match.Headers) > 0 {
		for _, header := range match.Headers {
			name := strings.ToLower(string(header.Name))
			name = strings.ReplaceAll(name, "-", "_")

			var this []v1.StringOrSlice
			this = append(this, v1.StringOrSlice{
				StrVal: "http_" + name,
			})

			switch *header.Type {
			case gatewayv1.HeaderMatchExact:
				this = append(this, v1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1.HeaderMatchRegularExpression:
				this = append(this, v1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown header match type " + string(*header.Type))
			}

			this = append(this, v1.StringOrSlice{
				StrVal: header.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if len(match.QueryParams) > 0 {
		for _, query := range match.QueryParams {
			var this []v1.StringOrSlice
			this = append(this, v1.StringOrSlice{
				StrVal: "arg_" + strings.ToLower(fmt.Sprintf("%v", query.Name)),
			})

			switch *query.Type {
			case gatewayv1.QueryParamMatchExact:
				this = append(this, v1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1.QueryParamMatchRegularExpression:
				this = append(this, v1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown query match type " + string(*query.Type))
			}

			this = append(this, v1.StringOrSlice{
				StrVal: query.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if match.Method != nil {
		route.Methods = []string{
			string(*match.Method),
		}
	}

	return route, nil
}
