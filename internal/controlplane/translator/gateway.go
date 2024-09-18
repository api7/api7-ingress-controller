package translator

import (
	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (t *Translator) TranslateGateway(tctx *TranslateContext, obj *gatewayv1.Gateway) (*TranslateResult, error) {
	result := &TranslateResult{}
	//set ssl
	for _, listener := range obj.Spec.Listeners {
		tctx.GatewayTLSConfig = append(tctx.GatewayTLSConfig, *listener.TLS)
		ssl, err := t.translateSecret(tctx, listener, obj.Name, obj.Namespace)
		if err != nil {
			return nil, err
		}
		result.SSL = append(result.SSL, ssl)
	}
	return result, nil
}

func (t *Translator) translateSecret(tctx *TranslateContext, listener gatewayv1.Listener, name, ns string) (*v1.Ssl, error) {
	if tctx.Secrets == nil {
		return nil, nil
	}
	sslObj := &v1.Ssl{}
	sslObj.ID = uuid.NewString()
	sslObj.Cert = string(tctx.Secrets[types.NamespacedName{Namespace: ns, Name: name}].Data["tls.crt"])
	if listener.Hostname != nil {
		sslObj.Snis = []string{string(*listener.Hostname)}
	}
	sslObj.Key = string(tctx.Secrets[types.NamespacedName{Namespace: ns, Name: name}].Data["tls.key"])
	return sslObj, nil
}
