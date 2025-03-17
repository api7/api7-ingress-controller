package translator

import (
	"encoding/json"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// fillPluginsFromGatewayProxy fill plugins from GatewayProxy to given plugins
func (t *Translator) fillPluginsFromGatewayProxy(plugins adctypes.Plugins, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}

	for _, plugin := range gatewayProxy.Spec.Plugins {
		// only apply enabled plugins
		if !plugin.Enabled {
			continue
		}

		pluginName := plugin.Name
		var pluginConfig map[string]any
		if err := json.Unmarshal(plugin.Config.Raw, &pluginConfig); err != nil {
			log.Errorw("gateway proxy plugin config unmarshal failed", zap.Error(err), zap.String("plugin", pluginName))
			continue
		}

		log.Debugw("fill plugin from gateway proxy", zap.String("plugin", pluginName), zap.Any("config", pluginConfig))
		plugins[pluginName] = pluginConfig
	}
}

func (t *Translator) fillPluginMetadataFromGatewayProxy(pluginMetadata adctypes.Plugins, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}
	for pluginName, plugin := range gatewayProxy.Spec.PluginMetadata {
		var pluginConfig map[string]any
		if err := json.Unmarshal(plugin.Raw, &pluginConfig); err != nil {
			log.Errorw("gateway proxy plugin_metadata unmarshal failed", zap.Error(err), zap.Any("plugin", pluginName), zap.String("config", string(plugin.Raw)))
			continue
		}
		log.Debugw("fill plugin_metadata for gateway proxy", zap.String("plugin", pluginName), zap.Any("config", pluginConfig))
		pluginMetadata[pluginName] = plugin
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
		t.fillPluginsFromGatewayProxy(globalRules, tctx.GatewayProxy)
		t.fillPluginMetadataFromGatewayProxy(pluginMetadata, tctx.GatewayProxy)
		result.GlobalRules = globalRules
		result.PluginMetadata = pluginMetadata
	}

	return result, nil
}
