package translator

import (
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"

	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

func (t *Translator) TranslateIngressClass(tctx *provider.TranslateContext, obj *networkingv1.IngressClass) (*TranslateResult, error) {
	result := &TranslateResult{}

	rk := provider.ResourceKind{
		Kind:      obj.Kind,
		Namespace: obj.Namespace,
		Name:      obj.Name,
	}
	gatewayProxy, ok := tctx.GatewayProxies[rk]
	if !ok {
		log.Debugw("no GatewayProxy found for IngressClass", zap.String("ingressclass", obj.Name))
		return result, nil
	}

	globalRules := adctypes.GlobalRule{
		Plugins: make(adctypes.Plugins),
	}
	pluginMetadata := adctypes.PluginMetadata{
		Plugins: make(adctypes.Plugins),
	}
	// apply plugins from GatewayProxy to global rules
	t.fillPluginsFromGatewayProxy(globalRules, &gatewayProxy)
	t.fillPluginMetadataFromGatewayProxy(pluginMetadata, &gatewayProxy)

	result.GlobalRules = globalRules
	result.PluginMetadata = pluginMetadata

	return result, nil
}
