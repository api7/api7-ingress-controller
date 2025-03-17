package translator

import (
	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/provider"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// FillPluginsFromGatewayProxy fill plugins from GatewayProxy to given plugins
func (t *Translator) FillPluginsFromGatewayProxy(plugins adctypes.Plugins, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}

	for _, plugin := range gatewayProxy.Spec.Plugins {
		// only apply enabled plugins
		if plugin.Enabled {
			plugins[plugin.Name] = plugin.Config
		}
	}
}

func (t *Translator) FillPluginMetadataFromGatewayProxy(pluginMetadata adctypes.Plugins, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}
	for k, v := range gatewayProxy.Spec.PluginMetadata {
		pluginMetadata[k] = v
	}
}

// TranslateGateway translate gateway to adc resources
func (t *Translator) TranslateGateway(tctx *provider.TranslateContext, gateway *gatewayv1.Gateway) (*TranslateResult, error) {
	result := &TranslateResult{}

	// if gateway has a GatewayProxy resource, create a global service and apply plugins
	if tctx.GatewayProxy != nil {
		var (
			globalRules    = adctypes.Plugins{}
			pluginMetadata = adctypes.Plugins{}
		)
		// apply plugins from GatewayProxy to global rules
		t.FillPluginsFromGatewayProxy(globalRules, tctx.GatewayProxy)
		t.FillPluginMetadataFromGatewayProxy(pluginMetadata, tctx.GatewayProxy)
		result.GlobalRules = globalRules
		result.PluginMetadata = pluginMetadata
	}

	return result, nil
}
