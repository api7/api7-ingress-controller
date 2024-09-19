package translator

import (
	"fmt"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/internal/controlplane/label"
	"github.com/api7/api7-ingress-controller/pkg/id"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (t *Translator) TranslateGateway(tctx *TranslateContext, obj *gatewayv1.Gateway) (*TranslateResult, error) {
	result := &TranslateResult{}
	for _, listener := range obj.Spec.Listeners {
		tctx.GatewayTLSConfig = append(tctx.GatewayTLSConfig, *listener.TLS)
		ssl, err := t.translateSecret(tctx, listener, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to translate secret: %w", err)
		}
		result.SSL = append(result.SSL, ssl)
	}
	return result, nil
}

func (t *Translator) translateSecret(tctx *TranslateContext, listener gatewayv1.Listener, obj *gatewayv1.Gateway) (*v1.Ssl, error) {
	if tctx.Secrets == nil {
		return nil, nil
	}
	sslObj := &v1.Ssl{}
	ns := obj.GetNamespace()
	name := obj.GetName()
	sslObj.ID = id.GenID(fmt.Sprintf("%s_%s_%s", ns, name, listener.Name))
	if listener.Hostname != nil && *listener.Hostname != "" {
		sslObj.Snis = []string{string(*listener.Hostname)}
	}
	secret := tctx.Secrets[types.NamespacedName{Namespace: ns, Name: name}]
	cert, key, err := extractKeyPair(secret, true)
	if err != nil {
		return nil, err
	}
	sslObj.Cert = string(cert)
	sslObj.Key = string(key)
	sslObj.Labels = label.GenLabel(obj)
	return sslObj, nil
}

func extractKeyPair(s *corev1.Secret, hasPrivateKey bool) ([]byte, []byte, error) {
	if _, ok := s.Data["cert"]; ok {
		return extractApisixSecretKeyPair(s, hasPrivateKey)
	} else if _, ok := s.Data[corev1.TLSCertKey]; ok {
		return extractKubeSecretKeyPair(s, hasPrivateKey)
	} else if ca, ok := s.Data[corev1.ServiceAccountRootCAKey]; ok && !hasPrivateKey {
		return ca, nil, nil
	} else {
		return nil, nil, errors.New("unknown secret format")
	}
}

func extractApisixSecretKeyPair(s *corev1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data["cert"]
	if !ok {
		return nil, nil, errors.New("missing cert field")
	}

	if hasPrivateKey {
		key, ok = s.Data["key"]
		if !ok {
			return nil, nil, errors.New("missing key field")
		}
	}
	return
}

func extractKubeSecretKeyPair(s *corev1.Secret, hasPrivateKey bool) (cert []byte, key []byte, err error) {
	var ok bool
	cert, ok = s.Data[corev1.TLSCertKey]
	if !ok {
		return nil, nil, errors.New("missing cert field")
	}

	if hasPrivateKey {
		key, ok = s.Data[corev1.TLSPrivateKeyKey]
		if !ok {
			return nil, nil, errors.New("missing key field")
		}
	}
	return
}
