// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package translator

import (
	"fmt"
	"reflect"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

type ResourceHandler interface {
	Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error)
}

type HTTPRouteHandler struct{ Translator *Translator }

func (h *HTTPRouteHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	hr := obj.(*gatewayv1.HTTPRoute)
	result, err := h.Translator.TranslateHTTPRoute(tctx, hr.DeepCopy())
	result.ResourceTypes = []string{"service"}
	return result, err
}

type GatewayHandler struct{ Translator *Translator }

func (h *GatewayHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	gw := obj.(*gatewayv1.Gateway)
	result, err := h.Translator.TranslateGateway(tctx, gw.DeepCopy())
	result.ResourceTypes = []string{"global_rule", "ssl", "plugin_metadata"}
	return result, err
}

type IngressHandler struct{ Translator *Translator }

func (h *IngressHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	ing := obj.(*networkingv1.Ingress)
	result, err := h.Translator.TranslateIngress(tctx, ing.DeepCopy())
	result.ResourceTypes = []string{"service", "ssl"}
	return result, err
}

type ConsumerV1alpha1Handler struct{ Translator *Translator }

func (h *ConsumerV1alpha1Handler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	c := obj.(*v1alpha1.Consumer)
	result, err := h.Translator.TranslateConsumerV1alpha1(tctx, c.DeepCopy())
	result.ResourceTypes = []string{"consumer"}
	return result, err
}

type IngressClassV1Handler struct{ Translator *Translator }

func (h *IngressClassV1Handler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	ic := obj.(*networkingv1.IngressClass)
	result, err := h.Translator.TranslateIngressClass(tctx, ic.DeepCopy())
	result.ResourceTypes = []string{"global_rule", "plugin_metadata"}
	return result, err
}

type IngressClassV1beta1Handler struct{ Translator *Translator }

func (h *IngressClassV1beta1Handler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	ic := pkgutils.ConvertToIngressClassV1(obj.(*networkingv1beta1.IngressClass).DeepCopy())
	result, err := h.Translator.TranslateIngressClass(tctx, ic)
	result.ResourceTypes = []string{"global_rule", "plugin_metadata"}
	return result, err
}

type ApisixRouteHandler struct{ Translator *Translator }

func (h *ApisixRouteHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	ar := obj.(*apiv2.ApisixRoute)
	result, err := h.Translator.TranslateApisixRoute(tctx, ar.DeepCopy())
	result.ResourceTypes = []string{"service"}
	return result, err
}

type ApisixGlobalRuleHandler struct{ Translator *Translator }

func (h *ApisixGlobalRuleHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	gr := obj.(*apiv2.ApisixGlobalRule)
	result, err := h.Translator.TranslateApisixGlobalRule(tctx, gr.DeepCopy())
	result.ResourceTypes = []string{"global_rule"}
	return result, err
}

type ApisixTlsHandler struct{ Translator *Translator }

func (h *ApisixTlsHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	tls := obj.(*apiv2.ApisixTls)
	result, err := h.Translator.TranslateApisixTls(tctx, tls.DeepCopy())
	result.ResourceTypes = []string{"ssl"}
	return result, err
}

type ApisixConsumerHandler struct{ Translator *Translator }

func (h *ApisixConsumerHandler) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	c := obj.(*apiv2.ApisixConsumer)
	result, err := h.Translator.TranslateApisixConsumer(tctx, c.DeepCopy())
	result.ResourceTypes = []string{"consumer"}
	return result, err
}

func (t *Translator) Register(obj client.Object, handler ResourceHandler) {
	t.register[reflect.TypeOf(obj)] = handler
}

func (t *Translator) Translate(tctx *provider.TranslateContext, obj client.Object) (*TranslateResult, error) {
	handler, ok := t.register[reflect.TypeOf(obj)]
	if !ok {
		return nil, fmt.Errorf("no handler registered for object type %s", reflect.TypeOf(obj))
	}
	return handler.Translate(tctx, obj)
}
