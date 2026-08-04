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
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	sslutils "github.com/apache/apisix-ingress-controller/internal/ssl"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (t *Translator) TranslateGateway(tctx *provider.TranslateContext, obj *gatewayv1.Gateway) (*TranslateResult, error) {
	result := &TranslateResult{}
	for _, listener := range obj.Spec.Listeners {
		if listener.TLS != nil {
			tctx.GatewayTLSConfig = append(tctx.GatewayTLSConfig, *listener.TLS)
			ssl, err := t.translateSecret(tctx, listener, obj)
			if err != nil {
				return nil, fmt.Errorf("failed to translate secret: %w", err)
			}
			result.SSL = append(result.SSL, ssl...)
		}
	}
	ssls, err := dedupGatewaySSLSNIs(result.SSL)
	if err != nil {
		return nil, err
	}
	result.SSL = ssls

	rk := utils.NamespacedNameKind(obj)
	gatewayProxy, ok := tctx.GatewayProxies[rk]
	if !ok {
		t.Log.V(1).Info("no GatewayProxy found for Gateway", "gateway", obj.Name)
		return result, nil
	}

	globalRules := make(adctypes.GlobalRule)
	pluginMetadata := make(adctypes.PluginMetadata)
	// apply plugins from GatewayProxy to global rules
	t.fillPluginsFromGatewayProxy(globalRules, &gatewayProxy)
	t.fillPluginMetadataFromGatewayProxy(pluginMetadata, &gatewayProxy)
	result.GlobalRules = globalRules
	result.PluginMetadata = pluginMetadata

	return result, nil
}

// dedupGatewaySSLSNIs enforces global SNI uniqueness across a Gateway's SSL objects.
//
// Several HTTPS listeners may share one certificate: a listener without a hostname
// derives its SNIs from the certificate SANs (e.g. "*.wildcard.org"), which can then
// collide with another listener that pins that hostname explicitly. The api7ee data
// plane keys SSL objects by SNI and rejects the whole sync with "SNI already exists"
// when the same SNI appears on two objects, leaving the Gateway (and every Gateway
// sharing its GatewayProxy) permanently un-Accepted.
//
// A collision may only be collapsed when both objects would serve the same TLS
// configuration, because the surviving object decides what every client connecting
// with that SNI is handed:
//
//   - the later object brings no certificate the owner does not already have, and
//     agrees on client validation: the SNI is simply dropped from it;
//   - both objects claim exactly the same SNIs and agree on client validation: their
//     certificate arrays are merged, so a listener pairing an RSA with an ECDSA
//     certificate keeps both. Merging is confined to this case: extending the owner's
//     certificates when it claims SNIs the other does not would serve those SNIs a
//     certificate that was never meant for them.
//
// Anything else asks for one SNI to be served with two different certificates or two
// different client policies, which the data plane cannot express. Keeping the first
// listener would hand the Gateway an arbitrary TLS configuration while its status
// still reports Accepted, so the conflict is reported instead and surfaces as
// Accepted=False on the Gateway.
func dedupGatewaySSLSNIs(ssls []*adctypes.SSL) ([]*adctypes.SSL, error) {
	if len(ssls) == 0 {
		return ssls, nil
	}
	// Claimed SNI sets are read after the owner has been trimmed, so the original
	// sets are captured up front.
	claims := make([]map[string]struct{}, len(ssls))
	for i, ssl := range ssls {
		claims[i] = sniSet(ssl.Snis)
	}

	owners := make(map[string]int)
	deduped := ssls[:0]
	for i, ssl := range ssls {
		kept := ssl.Snis[:0]
		for _, sni := range ssl.Snis {
			owner, claimed := owners[sni]
			if !claimed {
				owners[sni] = i
				kept = append(kept, sni)
				continue
			}
			if !clientClassEqual(ssls[owner].Client, ssl.Client) {
				return nil, fmt.Errorf("listeners disagree on the client certificate validation of SNI %q; "+
					"one SNI can only be served with a single TLS configuration", sni)
			}
			switch {
			// Nothing to carry over: the owner already serves these certificates.
			case certificatesContain(ssls[owner].Certificates, ssl.Certificates):
			// Same SNIs on both sides, so extra certificates are safe to merge.
			case sniSetEqual(claims[owner], claims[i]):
				ssls[owner].Certificates = mergeCertificates(ssls[owner].Certificates, ssl.Certificates)
			default:
				return nil, fmt.Errorf("listeners disagree on the certificate of SNI %q; "+
					"one SNI can only be served with a single TLS configuration", sni)
			}
		}
		ssl.Snis = kept
		if len(ssl.Snis) == 0 {
			continue
		}
		deduped = append(deduped, ssl)
	}
	return deduped, nil
}

func sniSet(snis []string) map[string]struct{} {
	set := make(map[string]struct{}, len(snis))
	for _, sni := range snis {
		set[sni] = struct{}{}
	}
	return set
}

func sniSetEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for sni := range a {
		if _, ok := b[sni]; !ok {
			return false
		}
	}
	return true
}

func clientClassEqual(a, b *adctypes.ClientClass) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.DeepEqual(*a, *b)
}

// certificatesContain reports whether every certificate of sub is already in super.
func certificatesContain(super, sub []adctypes.Certificate) bool {
	for _, cert := range sub {
		if !slices.Contains(super, cert) {
			return false
		}
	}
	return true
}

func mergeCertificates(into, extra []adctypes.Certificate) []adctypes.Certificate {
	for _, cert := range extra {
		if !slices.Contains(into, cert) {
			into = append(into, cert)
		}
	}
	return into
}

func (t *Translator) translateSecret(tctx *provider.TranslateContext, listener gatewayv1.Listener, obj *gatewayv1.Gateway) ([]*adctypes.SSL, error) {
	if tctx.Secrets == nil {
		return nil, nil
	}
	sslObjs := make([]*adctypes.SSL, 0)

	// TLS is terminated at the gateway unless the listener explicitly asks for
	// passthrough, in which case certificateRefs are not needed (and are ignored
	// per the Gateway API spec).
	mode := gatewayv1.TLSModeTerminate
	if listener.TLS.Mode != nil {
		mode = *listener.TLS.Mode
	}
	if mode == gatewayv1.TLSModePassthrough {
		return sslObjs, nil
	}

	if listener.TLS.CertificateRefs == nil {
		return nil, fmt.Errorf("no certificateRefs found in listener %s", listener.Name)
	}
	switch mode {
	case gatewayv1.TLSModeTerminate:
		// frontendValidation configures downstream mTLS: clients must present a
		// certificate signed by one of the referenced CAs during the TLS handshake.
		client, err := t.translateFrontendValidation(tctx, listener, obj)
		if err != nil {
			// The mode is unsupported on this listener only. Emit no SSL for it
			// (matching its Accepted=False status) and leave the rest of the
			// Gateway - other listeners, global rules, plugin metadata - intact.
			if errors.Is(err, errInsecureFallbackUnsupported) {
				t.Log.V(1).Info("skipping listener with unsupported frontendValidation mode",
					"gateway", obj.Name, "listener", listener.Name)
				return sslObjs, nil
			}
			return nil, err
		}
		for refIndex, ref := range listener.TLS.CertificateRefs {
			ns := obj.GetNamespace()
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			if listener.TLS.CertificateRefs[0].Kind != nil && *listener.TLS.CertificateRefs[0].Kind == internaltypes.KindSecret {
				sslObj := &adctypes.SSL{
					Snis: []string{},
				}
				name := listener.TLS.CertificateRefs[0].Name
				secretNN := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}
				secret := tctx.Secrets[secretNN]
				if secret == nil {
					continue
				}
				if secret.Data == nil {
					t.Log.Error(errors.New("secret data is nil"), "failed to get secret data", "secret", secretNN)
					return nil, fmt.Errorf("no secret data found for %s/%s", ns, name)
				}
				cert, key, err := sslutils.ExtractKeyPair(secret, true)
				if err != nil {
					t.Log.Error(err, "extract key pair", "secret", secretNN)
					return nil, err
				}
				sslObj.Certificates = append(sslObj.Certificates, adctypes.Certificate{
					Certificate: string(cert),
					Key:         string(key),
				})
				// we doesn't allow wildcard hostname
				if listener.Hostname != nil && *listener.Hostname != "" {
					sslObj.Snis = append(sslObj.Snis, string(*listener.Hostname))
				} else {
					hosts, err := sslutils.ExtractHostsFromCertificate(cert)
					if err != nil {
						return nil, err
					}
					if len(hosts) == 0 {
						t.Log.Info("no valid hostname found in certificate", "secret", secretNN.String())
						continue
					}
					sslObj.Snis = append(sslObj.Snis, hosts...)
				}
				sslObj.Client = client
				sslObj.ID = id.GenID(fmt.Sprintf("%s_%s_%d", adctypes.ComposeSSLName(internaltypes.KindGateway, obj.Namespace, obj.Name), listener.Name, refIndex))
				t.Log.V(1).Info("generated ssl id", "ssl id", sslObj.ID, "secret", secretNN.String())
				sslObj.Labels = label.GenLabel(obj)
				sslObjs = append(sslObjs, sslObj)
			}

		}
	default:
		return nil, fmt.Errorf("unknown TLS mode %s", mode)
	}

	return sslObjs, nil
}

// errInsecureFallbackUnsupported marks a listener whose frontendValidation asks for
// the Extended AllowInsecureFallback mode. It is handled in translateSecret by
// skipping that listener only; it must never fail the whole Gateway translation.
var errInsecureFallbackUnsupported = errors.New("frontendValidation mode AllowInsecureFallback is not supported")

// translateFrontendValidation builds the downstream mTLS client configuration from the
// Gateway's frontendValidation that applies to the listener. The referenced CA
// certificates (ConfigMap, key `ca.crt`) are bundled into a single trust anchor used
// to validate client certificates.
func (t *Translator) translateFrontendValidation(tctx *provider.TranslateContext, listener gatewayv1.Listener, obj *gatewayv1.Gateway) (*adctypes.ClientClass, error) {
	validation := internaltypes.FrontendTLSValidationForListener(obj, listener)
	if validation == nil || len(validation.CACertificateRefs) == 0 {
		return nil, nil
	}
	// APISIX can only enforce strict client-certificate verification (setting ca
	// turns on ssl_verify_client). AllowInsecureFallback ("accept even if the
	// client cert is missing or fails verification") cannot be expressed, so the
	// listener is reported Accepted=False/UnsupportedValue and programmed with no
	// SSL at all, rather than silently serving the opposite, strict behaviour.
	if validation.Mode == gatewayv1.AllowInsecureFallback {
		return nil, errInsecureFallbackUnsupported
	}

	cas := make([]string, 0, len(validation.CACertificateRefs))
	for _, ref := range validation.CACertificateRefs {
		// caCertificateRefs must be in the core API group. ConfigMap is the
		// Gateway API Core support; Secret is an implementation-specific extension.
		if ref.Group != "" && string(ref.Group) != corev1.GroupName {
			return nil, fmt.Errorf("unsupported frontendValidation caCertificateRef group %q in listener %s, only the core group is supported", ref.Group, listener.Name)
		}
		ns := obj.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		nn := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}

		kind := internaltypes.KindConfigMap
		if ref.Kind != "" {
			kind = string(ref.Kind)
		}
		var (
			ca  []byte
			err error
		)
		switch kind {
		case internaltypes.KindConfigMap:
			cm := tctx.ConfigMaps[nn]
			if cm == nil {
				return nil, fmt.Errorf("frontendValidation CA ConfigMap %s not found", nn.String())
			}
			if ca, err = sslutils.ExtractCAFromConfigMap(cm); err != nil {
				t.Log.Error(err, "failed to extract CA from configmap", "configmap", nn.String())
				return nil, fmt.Errorf("failed to extract CA from ConfigMap %s: %w", nn.String(), err)
			}
		case internaltypes.KindSecret:
			secret := tctx.Secrets[nn]
			if secret == nil {
				return nil, fmt.Errorf("frontendValidation CA Secret %s not found", nn.String())
			}
			if ca, err = sslutils.ExtractCAFromSecret(secret); err != nil {
				t.Log.Error(err, "failed to extract CA from secret", "secret", nn.String())
				return nil, fmt.Errorf("failed to extract CA from Secret %s: %w", nn.String(), err)
			}
		default:
			return nil, fmt.Errorf("unsupported frontendValidation caCertificateRef kind %q in listener %s, only ConfigMap and Secret are supported", ref.Kind, listener.Name)
		}
		cas = append(cas, strings.TrimSpace(string(ca)))
	}

	return &adctypes.ClientClass{
		CA: strings.Join(cas, "\n"),
	}, nil
}

// fillPluginsFromGatewayProxy fill plugins from GatewayProxy to given plugins
func (t *Translator) fillPluginsFromGatewayProxy(plugins adctypes.GlobalRule, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}

	for _, plugin := range gatewayProxy.Spec.Plugins {
		// only apply enabled plugins
		if !plugin.Enabled {
			continue
		}

		pluginName := plugin.Name
		pluginConfig := map[string]any{}
		if len(plugin.Config.Raw) > 0 {
			if err := json.Unmarshal(plugin.Config.Raw, &pluginConfig); err != nil {
				t.Log.Error(err, "gateway proxy plugin config unmarshal failed", "plugin", pluginName)
				continue
			}
		}
		plugins[pluginName] = pluginConfig
	}
	t.Log.V(1).Info("fill plugins for gateway proxy", "plugins", plugins)
}

func (t *Translator) fillPluginMetadataFromGatewayProxy(pluginMetadata adctypes.PluginMetadata, gatewayProxy *v1alpha1.GatewayProxy) {
	if gatewayProxy == nil {
		return
	}
	for pluginName, plugin := range gatewayProxy.Spec.PluginMetadata {
		var pluginConfig map[string]any
		if err := json.Unmarshal(plugin.Raw, &pluginConfig); err != nil {
			t.Log.Error(err, "gateway proxy plugin_metadata unmarshal failed", "plugin", pluginName, "config", string(plugin.Raw))
			continue
		}
		t.Log.V(1).Info("fill plugin_metadata for gateway proxy", "plugin", pluginName, "config", pluginConfig)
		pluginMetadata[pluginName] = pluginConfig
	}
}
