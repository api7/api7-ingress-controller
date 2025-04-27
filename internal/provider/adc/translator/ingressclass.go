package translator

import (
	"github.com/api7/api7-ingress-controller/internal/provider"
	networkingv1 "k8s.io/api/networking/v1"
)

func (t *Translator) TranslateIngressClass(tctx *provider.TranslateContext, obj *networkingv1.IngressClass) (*TranslateResult, error) {
	result := &TranslateResult{}
	return result, nil
}
