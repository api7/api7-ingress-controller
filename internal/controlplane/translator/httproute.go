package translator

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/internal/controlplane/label"
	"github.com/api7/api7-ingress-controller/internal/id"
)

func (t *Translator) translateEndpointSlice(endpointSlices []discoveryv1.EndpointSlice, weight int) v1.UpstreamNodes {
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
						Weight: weight,
					}
					nodes = append(nodes, node)
				}
			}
		}
	}

	return nodes
}

func (t *Translator) translateBackendRef(tctx *TranslateContext, ref gatewayv1.BackendRef) *v1.Upstream {
	var upstream v1.Upstream
	endpointSlices := tctx.EndpointSlices[types.NamespacedName{
		Namespace: string(*ref.Namespace),
		Name:      string(ref.Name),
	}]

	weight := 100
	if ref.Weight != nil {
		weight = int(*ref.Weight)
	}
	upstream.Nodes = t.translateEndpointSlice(endpointSlices, weight)
	return &upstream
}

func (t *Translator) TranslateGatewayHTTPRoute(tctx *TranslateContext, httpRoute *gatewayv1.HTTPRoute) (*TranslateResult, error) {
	result := &TranslateResult{}

	hosts := make([]string, 0, len(httpRoute.Spec.Hostnames))
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	rules := httpRoute.Spec.Rules

	for i, rule := range rules {
		backends := rule.BackendRefs
		if len(backends) == 0 {
			continue
		}

		upstreams := []*v1.Upstream{}
		for _, backend := range backends {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(httpRoute.Namespace)
				backend.Namespace = &namespace
			}
			upstream := t.translateBackendRef(tctx, backend.BackendRef)
			upstreams = append(upstreams, upstream)
		}

		var service v1.Service
		if len(upstreams) > 0 {
			service.Upstream = *upstreams[0]
			// TODO: support multiple upstreams
		}
		service.Name = v1.ComposeServiceNameWithRule(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d", i))
		service.ID = id.GenID(service.Name)
		service.Labels = label.GenLabel(httpRoute)
		service.Hosts = hosts
		result.Services = append(result.Services, &service)

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

	if match.Headers != nil && len(match.Headers) > 0 {
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

	if match.QueryParams != nil && len(match.QueryParams) > 0 {
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
