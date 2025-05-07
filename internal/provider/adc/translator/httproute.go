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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/gopkg/pkg/log"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/controller/label"
	"github.com/api7/api7-ingress-controller/internal/id"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

func (t *Translator) fillPluginsFromHTTPRouteFilters(
	plugins adctypes.Plugins,
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

func (t *Translator) fillPluginFromExtensionRef(plugins adctypes.Plugins, namespace string, extensionRef *gatewayv1.LocalObjectReference, tctx *provider.TranslateContext) {
	if extensionRef == nil {
		return
	}
	if extensionRef.Kind == "PluginConfig" {
		pluginconfig := tctx.PluginConfigs[types.NamespacedName{
			Namespace: namespace,
			Name:      string(extensionRef.Name),
		}]
		if pluginconfig == nil {
			return
		}
		for _, plugin := range pluginconfig.Spec.Plugins {
			pluginName := plugin.Name
			var pluginconfig map[string]any
			if len(plugin.Config.Raw) > 0 {
				if err := json.Unmarshal(plugin.Config.Raw, &pluginconfig); err != nil {
					log.Errorw("plugin config unmarshal failed", zap.Error(err))
					continue
				}
			}
			plugins[pluginName] = pluginconfig
		}
		log.Debugw("fill plugin from extension ref", zap.Any("plugins", plugins))
	}
}

func (t *Translator) fillPluginFromURLRewriteFilter(plugins adctypes.Plugins, urlRewrite *gatewayv1.HTTPURLRewriteFilter, matches []gatewayv1.HTTPRouteMatch) {
	pluginName := adctypes.PluginProxyRewrite
	obj := plugins[pluginName]
	var plugin *adctypes.RewriteConfig
	if obj == nil {
		plugin = &adctypes.RewriteConfig{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*adctypes.RewriteConfig)
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

func (t *Translator) fillPluginFromHTTPRequestHeaderFilter(plugins adctypes.Plugins, reqHeaderModifier *gatewayv1.HTTPHeaderFilter) {
	pluginName := adctypes.PluginProxyRewrite
	obj := plugins[pluginName]
	var plugin *adctypes.RewriteConfig
	if obj == nil {
		plugin = &adctypes.RewriteConfig{
			Headers: &adctypes.Headers{
				Add:    make(map[string]string, len(reqHeaderModifier.Add)),
				Set:    make(map[string]string, len(reqHeaderModifier.Set)),
				Remove: make([]string, 0, len(reqHeaderModifier.Remove)),
			},
		}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*adctypes.RewriteConfig)
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

func (t *Translator) fillPluginFromHTTPResponseHeaderFilter(plugins adctypes.Plugins, respHeaderModifier *gatewayv1.HTTPHeaderFilter) {
	pluginName := adctypes.PluginResponseRewrite
	obj := plugins[pluginName]
	var plugin *adctypes.ResponseRewriteConfig
	if obj == nil {
		plugin = &adctypes.ResponseRewriteConfig{
			Headers: &adctypes.ResponseHeaders{
				Add:    make([]string, 0, len(respHeaderModifier.Add)),
				Set:    make(map[string]string, len(respHeaderModifier.Set)),
				Remove: make([]string, 0, len(respHeaderModifier.Remove)),
			},
		}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*adctypes.ResponseRewriteConfig)
	}
	for _, header := range respHeaderModifier.Add {
		plugin.Headers.Add = append(plugin.Headers.Add, fmt.Sprintf("%s: %s", header.Name, header.Value))
	}
	for _, header := range respHeaderModifier.Set {
		plugin.Headers.Set[string(header.Name)] = header.Value
	}
	plugin.Headers.Remove = append(plugin.Headers.Remove, respHeaderModifier.Remove...)
}

func (t *Translator) fillPluginFromHTTPRequestMirrorFilter(plugins adctypes.Plugins, namespace string, reqMirror *gatewayv1.HTTPRequestMirrorFilter) {
	pluginName := adctypes.PluginProxyMirror
	obj := plugins[pluginName]

	var plugin *adctypes.RequestMirror
	if obj == nil {
		plugin = &adctypes.RequestMirror{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*adctypes.RequestMirror)
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

func (t *Translator) fillPluginFromHTTPRequestRedirectFilter(plugins adctypes.Plugins, reqRedirect *gatewayv1.HTTPRequestRedirectFilter) {
	pluginName := adctypes.PluginRedirect
	obj := plugins[pluginName]

	var plugin *adctypes.RedirectConfig
	if obj == nil {
		plugin = &adctypes.RedirectConfig{}
		plugins[pluginName] = plugin
	} else {
		plugin = obj.(*adctypes.RedirectConfig)
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

func (t *Translator) fillHTTPRoutePoliciesForHTTPRoute(tctx *provider.TranslateContext, routes []*adctypes.Route, rule gatewayv1.HTTPRouteRule) {
	var policies []v1alpha1.HTTPRoutePolicy
	for _, policy := range tctx.HTTPRoutePolicies {
		for _, ref := range policy.Spec.TargetRefs {
			if string(ref.Kind) == "HTTPRoute" && (ref.SectionName == nil || *ref.SectionName == "" || ptr.Equal(ref.SectionName, rule.Name)) {
				policies = append(policies, policy)
				break
			}
		}
	}

	t.fillHTTPRoutePolicies(routes, policies)
}

func (t *Translator) fillHTTPRoutePoliciesForIngress(tctx *provider.TranslateContext, routes []*adctypes.Route) {
	t.fillHTTPRoutePolicies(routes, tctx.HTTPRoutePolicies)
}

func (t *Translator) fillHTTPRoutePolicies(routes []*adctypes.Route, policies []v1alpha1.HTTPRoutePolicy) {
	for _, policy := range policies {
		for _, route := range routes {
			route.Priority = policy.Spec.Priority
			for _, data := range policy.Spec.Vars {
				var v []adctypes.StringOrSlice
				if err := json.Unmarshal(data.Raw, &v); err != nil {
					log.Errorf("failed to unmarshal spec.Vars item to []StringOrSlice, data: %s", string(data.Raw)) // todo: update status
					continue
				}
				route.Vars = append(route.Vars, v)
			}
		}
	}
}

func (t *Translator) translateEndpointSlice(portName *string, weight int, endpointSlices []discoveryv1.EndpointSlice) adctypes.UpstreamNodes {
	var nodes adctypes.UpstreamNodes
	if len(endpointSlices) == 0 {
		return nodes
	}
	for _, endpointSlice := range endpointSlices {
		for _, port := range endpointSlice.Ports {
			if portName != nil && !ptr.Equal(portName, port.Name) {
				continue
			}
			for _, endpoint := range endpointSlice.Endpoints {
				for _, addr := range endpoint.Addresses {
					node := adctypes.UpstreamNode{
						Host:   addr,
						Port:   int(*port.Port),
						Weight: weight,
					}
					nodes = append(nodes, node)
				}
			}
			if portName != nil {
				break
			}
		}
	}

	return nodes
}

func (t *Translator) translateBackendRef(tctx *provider.TranslateContext, ref gatewayv1.BackendRef) adctypes.UpstreamNodes {
	key := types.NamespacedName{
		Namespace: string(*ref.Namespace),
		Name:      string(ref.Name),
	}
	service, ok := tctx.Services[key]
	if !ok {
		return adctypes.UpstreamNodes{}
	}

	weight := 1
	if ref.Weight != nil {
		weight = int(*ref.Weight)
	}

	if service.Spec.Type == corev1.ServiceTypeExternalName {
		port := 80
		if ref.Port != nil {
			port = int(*ref.Port)
		}
		return adctypes.UpstreamNodes{
			{
				Host:   service.Spec.ExternalName,
				Port:   port,
				Weight: weight,
			},
		}
	}

	var portName *string
	if ref.Port != nil {
		for _, p := range service.Spec.Ports {
			if int(p.Port) == int(*ref.Port) {
				portName = ptr.To(p.Name)
				break
			}
		}
		if portName == nil {
			return adctypes.UpstreamNodes{}
		}
	}

	endpointSlices := tctx.EndpointSlices[key]
	return t.translateEndpointSlice(portName, weight, endpointSlices)
}

// calculateMatchPriority calculate the priority of HTTPRouteMatch, according to the Gateway API specification
// the higher the return value, the higher the priority
func calculateMatchPriority(match *gatewayv1.HTTPRouteMatch) int64 {
	var score int64 = 0

	// 1. Exact path matches have the highest priority
	if match.Path != nil && match.Path.Type != nil && *match.Path.Type == gatewayv1.PathMatchExact {
		score += 10000
	} else if match.Path != nil && match.Path.Type != nil && *match.Path.Type == gatewayv1.PathMatchPathPrefix && match.Path.Value != nil {
		// 2. Prefix path matches, the longer the string, the higher the priority
		score += 1000 + int64(len(*match.Path.Value))
	}

	// 3. Method matching
	if match.Method != nil {
		score += 100
	}

	// 4. Header matching, the more headers, the higher the priority
	score += int64(len(match.Headers) * 10)

	// 5. Query parameter matching, the more query parameters, the higher the priority
	score += int64(len(match.QueryParams))

	return score
}

func (t *Translator) TranslateHTTPRoute(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) (*TranslateResult, error) {
	result := &TranslateResult{}

	hosts := make([]string, 0, len(httpRoute.Spec.Hostnames))
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	rules := httpRoute.Spec.Rules

	labels := label.GenLabel(httpRoute)

	for i, rule := range rules {
		upstream := adctypes.NewDefaultUpstream()
		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(httpRoute.Namespace)
				backend.Namespace = &namespace
			}
			upNodes := t.translateBackendRef(tctx, backend.BackendRef)
			t.AttachBackendTrafficPolicyToUpstream(backend.BackendRef, tctx.BackendTrafficPolicies, upstream)
			upstream.Nodes = append(upstream.Nodes, upNodes...)
		}
		t.attachBackendTrafficPolicyToUpstream(nil, upstream)

		// todo: support multiple backends
		service := adctypes.NewDefaultService()
		service.Labels = labels

		service.Name = adctypes.ComposeServiceNameWithRule(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d", i))
		service.ID = id.GenID(service.Name)
		service.Hosts = hosts
		service.Upstream = upstream
		t.fillPluginsFromHTTPRouteFilters(service.Plugins, httpRoute.GetNamespace(), rule.Filters, rule.Matches, tctx)

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
		} else {
			// Sort the matches by priority
			sort.Slice(matches, func(a, b int) bool {
				return calculateMatchPriority(&matches[a]) > calculateMatchPriority(&matches[b])
			})
		}

		routes := []*adctypes.Route{}
		for j, match := range matches {
			route, err := t.translateGatewayHTTPRouteMatch(&match)
			if err != nil {
				return nil, err
			}

			name := adctypes.ComposeRouteName(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			route.Name = name
			route.ID = id.GenID(name)
			route.Labels = labels
			route.EnableWebsocket = ptr.To(true)

			// Set the route priority
			priority := calculateMatchPriority(&match)
			route.Priority = &priority

			routes = append(routes, route)
		}
		t.fillHTTPRoutePoliciesForHTTPRoute(tctx, routes, rule)
		service.Routes = routes

		result.Services = append(result.Services, service)
	}

	return result, nil
}

// NOTE: Dashboard not support Vars, matches only support Path, not support Headers, QueryParams
func (t *Translator) translateGatewayHTTPRouteMatch(match *gatewayv1.HTTPRouteMatch) (*adctypes.Route, error) {
	route := &adctypes.Route{}

	if match.Path != nil {
		switch *match.Path.Type {
		case gatewayv1.PathMatchExact:
			route.Uris = []string{*match.Path.Value}
		case gatewayv1.PathMatchPathPrefix:
			pathValue := *match.Path.Value
			route.Uris = []string{pathValue}

			if strings.HasSuffix(pathValue, "/") {
				route.Uris = append(route.Uris, pathValue+"*")
			} else {
				route.Uris = append(route.Uris, pathValue+"/*")
			}
		case gatewayv1.PathMatchRegularExpression:
			var this []adctypes.StringOrSlice
			this = append(this, adctypes.StringOrSlice{
				StrVal: "uri",
			})
			this = append(this, adctypes.StringOrSlice{
				StrVal: "~~",
			})
			this = append(this, adctypes.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		default:
			return nil, errors.New("unknown path match type " + string(*match.Path.Type))
		}
	} else {
		/* If no matches are specified, the default is a prefix
		path match on "/", which has the effect of matching every
		HTTP request. */
		route.Uris = []string{"/", "/*"}
	}

	if len(match.Headers) > 0 {
		for _, header := range match.Headers {
			name := strings.ToLower(string(header.Name))
			name = strings.ReplaceAll(name, "-", "_")

			var this []adctypes.StringOrSlice
			this = append(this, adctypes.StringOrSlice{
				StrVal: "http_" + name,
			})

			matchType := gatewayv1.HeaderMatchExact
			if header.Type != nil {
				matchType = *header.Type
			}

			switch matchType {
			case gatewayv1.HeaderMatchExact:
				this = append(this, adctypes.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1.HeaderMatchRegularExpression:
				this = append(this, adctypes.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown header match type " + string(matchType))
			}

			this = append(this, adctypes.StringOrSlice{
				StrVal: header.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if len(match.QueryParams) > 0 {
		for _, query := range match.QueryParams {
			var this []adctypes.StringOrSlice
			this = append(this, adctypes.StringOrSlice{
				StrVal: "arg_" + strings.ToLower(fmt.Sprintf("%v", query.Name)),
			})

			queryType := gatewayv1.QueryParamMatchExact
			if query.Type != nil {
				queryType = *query.Type
			}

			switch queryType {
			case gatewayv1.QueryParamMatchExact:
				this = append(this, adctypes.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1.QueryParamMatchRegularExpression:
				this = append(this, adctypes.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown query match type " + string(queryType))
			}

			this = append(this, adctypes.StringOrSlice{
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
