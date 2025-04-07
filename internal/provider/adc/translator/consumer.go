package translator

import (
	"encoding/json"

	"github.com/api7/api7-ingress-controller/api/adc"
	adctypes "github.com/api7/api7-ingress-controller/api/adc"
	"github.com/api7/api7-ingress-controller/api/v1alpha1"
	"github.com/api7/api7-ingress-controller/internal/provider"
	"k8s.io/apimachinery/pkg/types"
)

func (t *Translator) TranslateConsumerV1alpha1(tctx *provider.TranslateContext, consumerV *v1alpha1.Consumer) (*TranslateResult, error) {
	result := &TranslateResult{}
	if consumerV == nil {
		return result, nil
	}

	username := adctypes.ComposeConsumerName(consumerV.Namespace, consumerV.Name)
	consumer := &adctypes.Consumer{
		Username: username,
	}
	credentials := make([]adctypes.Credential, 0, len(consumerV.Spec.Credentials))
	for _, credentialSpec := range consumerV.Spec.Credentials {
		credential := adc.Credential{}
		credential.Name = credentialSpec.Name
		credential.Type = credentialSpec.Type
		if credentialSpec.SecretRef != nil {
			ns := consumerV.Namespace
			if credentialSpec.SecretRef.Namespace != nil {
				ns = *credentialSpec.SecretRef.Namespace
			}
			secret := tctx.Secrets[types.NamespacedName{
				Namespace: ns,
				Name:      credentialSpec.SecretRef.Name,
			}]
			if secret == nil {
				continue
			}
			authConfig := make(map[string]any)
			for k, v := range secret.Data {
				authConfig[k] = string(v)
			}
			credential.Config = authConfig
		} else {
			authConfig := make(map[string]any)
			if err := json.Unmarshal(credentialSpec.Config.Raw, &authConfig); err != nil {
				t.Log.Error(err, "failed to unmarshal credential config", "credential", credentialSpec)
				continue
			}
			credential.Config = authConfig
		}
		credentials = append(credentials, credential)
	}
	consumer.Credentials = credentials

	plugins := adctypes.Plugins{}
	for _, plugin := range consumerV.Spec.Plugins {
		pluginName := plugin.Name
		pluginConfig := make(map[string]any)
		if err := json.Unmarshal(plugin.Config.Raw, &pluginConfig); err != nil {
			t.Log.Error(err, "failed to unmarshal plugin config", "plugin", plugin)
			continue
		}
		plugins[pluginName] = pluginConfig
	}
	consumer.Plugins = plugins
	result.Consumers = append(result.Consumers, consumer)
	return result, nil
}
