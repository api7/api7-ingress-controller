package translator

import (
	"github.com/api7/api7-ingress-controller/internal/provider"
	networkingv1 "k8s.io/api/networking/v1"
)

func (t *Translator) TranslateIngress(tctx *provider.TranslateContext, obj *networkingv1.Ingress) (*TranslateResult, error) {
	return nil, nil
}
