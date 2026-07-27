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
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const testCACert = `-----BEGIN CERTIFICATE-----
MIIBQzCB6qADAgECAgEBMAoGCCqGSM49BAMCMBIxEDAOBgNVBAMTB3Rlc3QtY2Ew
HhcNNzAwMTAxMDAwMDAwWhcNMzgwMTE5MDMxNDA4WjASMRAwDgYDVQQDEwd0ZXN0
LWNhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEJo4AsM30ZHN+mYeHjqwceGBz
V2bMz1+OyNXuaPYVrSF7HShZhanOYNHb6QLNhjGxMsBDQHVLolPjyTQJp9R5GqMx
MC8wDgYDVR0PAQH/BAQDAgIEMB0GA1UdDgQWBBRzjh0YVmnpN/cFJziO0aYySuti
4DAKBggqhkjOPQQDAgNIADBFAiEA7fEGiQA7wX0LrrkRH4KplAPOgVV5Kvm/1dv1
3TLq9ssCIHKkv2dhydRvv36KC1WsRDcrl7W+7YmEnCS9PZfb8agM
-----END CERTIFICATE-----`

func newTLSGateway(frontendValidation *gatewayv1.FrontendTLSValidation) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "gw",
		},
		Spec: gatewayv1.GatewaySpec{
			// In Gateway API v1.6 frontendValidation is declared at the Gateway level.
			TLS: &gatewayv1.GatewayTLSConfig{
				Frontend: &gatewayv1.FrontendTLSConfig{
					Default: gatewayv1.TLSConfig{
						Validation: frontendValidation,
					},
				},
			},
			Listeners: []gatewayv1.Listener{
				{
					Name:     "https",
					Protocol: gatewayv1.HTTPSProtocolType,
					Hostname: ptr.To(gatewayv1.Hostname("example.com")),
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayv1.TLSModeTerminate),
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{
								Kind: ptr.To(gatewayv1.Kind("Secret")),
								Name: gatewayv1.ObjectName("server-cert"),
							},
						},
					},
				},
			},
		},
	}
}

// TestTranslateGateway_InsecureFallbackIsContained pins the blast radius of an
// unsupported frontendValidation mode: it must cost only the listener that asked
// for it, not the whole Gateway. A per-port override is the cheapest way for a
// user to trip this, and returning an error here would abort TranslateGateway and
// stop every SSL object, global rule and plugin metadata for the Gateway.
func TestTranslateGateway_InsecureFallbackIsContained(t *testing.T) {
	tr := &Translator{Log: logr.Discard()}
	gateway := newTLSGateway(nil)
	gateway.Spec.TLS.Frontend.PerPort = []gatewayv1.TLSPortConfig{
		{
			Port: gatewayv1.PortNumber(8443),
			TLS: gatewayv1.TLSConfig{
				Validation: &gatewayv1.FrontendTLSValidation{
					Mode: gatewayv1.AllowInsecureFallback,
					CACertificateRefs: []gatewayv1.ObjectReference{
						{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
					},
				},
			},
		},
	}
	// A second HTTPS listener on the overridden port; the first one stays on 443.
	unsupported := *gateway.Spec.Listeners[0].DeepCopy()
	unsupported.Name = "https-8443"
	unsupported.Port = gatewayv1.PortNumber(8443)
	unsupported.Hostname = ptr.To(gatewayv1.Hostname("fallback.example.com"))
	gateway.Spec.Listeners = append(gateway.Spec.Listeners, unsupported)

	result, err := tr.TranslateGateway(newTranslateContextWithTLS(), gateway)
	require.NoError(t, err, "one unsupported listener must not fail the Gateway")
	require.NotNil(t, result)
	// Only the listener on 443 is programmed; the 8443 one contributes nothing.
	require.Len(t, result.SSL, 1)
	assert.Contains(t, result.SSL[0].Snis, "example.com")
	assert.NotContains(t, result.SSL[0].Snis, "fallback.example.com")
}

func newTranslateContextWithTLS() *provider.TranslateContext {
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "server-cert"}] = &corev1.Secret{
		Data: map[string][]byte{
			"cert": []byte("server-cert-data"),
			"key":  []byte("server-key-data"),
		},
	}
	tctx.ConfigMaps[types.NamespacedName{Namespace: "default", Name: "ca-cm"}] = &corev1.ConfigMap{
		Data: map[string]string{
			corev1.ServiceAccountRootCAKey: testCACert,
		},
	}
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "ca-secret"}] = &corev1.Secret{
		Data: map[string][]byte{
			corev1.ServiceAccountRootCAKey: []byte(testCACert),
		},
	}
	return tctx
}

// wildcardSANCert has SubjectAltName DNS entries "*", "*.org" and "*.wildcard.org",
// mirroring the Gateway API conformance "tls-validity-checks-certificate".
const wildcardSANCert = `-----BEGIN CERTIFICATE-----
MIIBdzCCAR6gAwIBAgIUbQBIRhcj+CxDYVUGRnyjp1U3aOIwCgYIKoZIzj0EAwIw
GDEWMBQGA1UEAwwNd2lsZGNhcmQtdGVzdDAeFw0yNjA3MjIyMDAzMzNaFw0zNjA3
MTkyMDAzMzNaMBgxFjAUBgNVBAMMDXdpbGRjYXJkLXRlc3QwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAARODu9OY/689fZmpgEJoUWVvjy7kjhkspOw6kb47oNscy7P
QsHrHpIIffRZhPHMmz2wzI14ESdTH6GwZO9uatNfo0YwRDAjBgNVHREEHDAaggEq
ggUqLm9yZ4IOKi53aWxkY2FyZC5vcmcwHQYDVR0OBBYEFBFwpVJptvLRFA6DCf7B
houdxXdoMAoGCCqGSM49BAMCA0cAMEQCIC8nLcK2Cpd9OB/P/phcIf8/ugeGetar
vy5jpEgv0MmVAiB4Mhxjtl3b3r1JNUXpqyUes/h1qlGJLh7vMUvZUEBocQ==
-----END CERTIFICATE-----`

const wildcardSANKey = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQghT8xLzsTLnWeNEv1
iUOosi/kieSfR2owSGL4BV1vAV+hRANCAARODu9OY/689fZmpgEJoUWVvjy7kjhk
spOw6kb47oNscy7PQsHrHpIIffRZhPHMmz2wzI14ESdTH6GwZO9uatNf
-----END PRIVATE KEY-----`

// TestTranslateGateway_DuplicateSNIAcrossListeners reproduces the Gateway API 1.6
// conformance base gateway "same-namespace-with-https-listener": several HTTPS
// listeners share one wildcard certificate. The listener without a hostname derives
// its SNIs from the certificate SANs (here "*.org", "*.wildcard.org"), which then
// collides with the explicit "*.wildcard.org" listener. The api7ee data plane
// enforces global SNI uniqueness and rejects the sync ("SNI already exists"), so the
// translator must not emit the same SNI on more than one SSL object.
func TestTranslateGateway_DuplicateSNIAcrossListeners(t *testing.T) {
	tr := &Translator{Log: logr.Discard()}
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "wildcard-cert"}] = &corev1.Secret{
		Data: map[string][]byte{
			"cert": []byte(wildcardSANCert),
			"key":  []byte(wildcardSANKey),
		},
	}

	certRef := []gatewayv1.SecretObjectReference{{
		Kind: ptr.To(gatewayv1.Kind("Secret")),
		Name: gatewayv1.ObjectName("wildcard-cert"),
	}}
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "https",
					Port:     443,
					Protocol: gatewayv1.HTTPSProtocolType,
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode:            ptr.To(gatewayv1.TLSModeTerminate),
						CertificateRefs: certRef,
					},
				},
				{
					Name:     "https-with-wildcard-hostname",
					Port:     443,
					Hostname: ptr.To(gatewayv1.Hostname("*.wildcard.org")),
					Protocol: gatewayv1.HTTPSProtocolType,
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode:            ptr.To(gatewayv1.TLSModeTerminate),
						CertificateRefs: certRef,
					},
				},
			},
		},
	}

	result, err := tr.TranslateGateway(tctx, gateway)
	require.NoError(t, err)

	seen := map[string]string{}
	for _, ssl := range result.SSL {
		for _, sni := range ssl.Snis {
			if prev, ok := seen[sni]; ok {
				t.Errorf("SNI %q appears on more than one SSL object (%s and %s); the api7ee data plane rejects duplicate SNIs", sni, prev, ssl.ID)
			}
			seen[sni] = ssl.ID
		}
	}
}

// TestDedupGatewaySSLSNIs covers what happens to the losing side of an SNI
// collision: only a configuration the surviving SSL object already serves may be
// dropped, and only a listener claiming exactly the same SNIs may contribute extra
// certificates. Everything else would silently hand the Gateway one listener's TLS
// configuration while the other listener still looks accepted.
func TestDedupGatewaySSLSNIs(t *testing.T) {
	var (
		certA = adctypes.Certificate{Certificate: "cert-a", Key: "key-a"}
		certB = adctypes.Certificate{Certificate: "cert-b", Key: "key-b"}
		caOne = &adctypes.ClientClass{CA: "ca-one"}
		caTwo = &adctypes.ClientClass{CA: "ca-two"}
		ssl   = func(client *adctypes.ClientClass, certs []adctypes.Certificate, snis ...string) *adctypes.SSL {
			return &adctypes.SSL{Certificates: certs, Client: client, Snis: snis}
		}
	)

	tests := []struct {
		name     string
		ssls     []*adctypes.SSL
		wantSnis [][]string
		wantCert [][]adctypes.Certificate
		wantErr  string
	}{
		{
			// The conformance case: one shared certificate, the hostname-less
			// listener derives "*.wildcard.org" from the SANs.
			name: "shared certificate keeps the first claimant",
			ssls: []*adctypes.SSL{
				ssl(nil, []adctypes.Certificate{certA}, "*.org", "*.wildcard.org"),
				ssl(nil, []adctypes.Certificate{certA}, "*.wildcard.org"),
			},
			wantSnis: [][]string{{"*.org", "*.wildcard.org"}},
			wantCert: [][]adctypes.Certificate{{certA}},
		},
		{
			// An RSA and an ECDSA certificate for the same hostname: both are
			// programmed, because neither listener serves an SNI the other does not.
			name: "same SNIs merge their certificates",
			ssls: []*adctypes.SSL{
				ssl(nil, []adctypes.Certificate{certA}, "example.com"),
				ssl(nil, []adctypes.Certificate{certB}, "example.com"),
			},
			wantSnis: [][]string{{"example.com"}},
			wantCert: [][]adctypes.Certificate{{certA, certB}},
		},
		{
			// Merging here would serve "*.org" a certificate meant only for
			// "*.wildcard.org", so the disagreement is reported instead.
			name: "differing certificate on overlapping SNIs conflicts",
			ssls: []*adctypes.SSL{
				ssl(nil, []adctypes.Certificate{certA}, "*.org", "*.wildcard.org"),
				ssl(nil, []adctypes.Certificate{certB}, "*.wildcard.org"),
			},
			wantErr: `disagree on the certificate of SNI "*.wildcard.org"`,
		},
		{
			// The certificate is the same, but one listener requires a client
			// certificate from a different CA. Only one of the two policies can be
			// programmed, and picking by listener order would be arbitrary.
			name: "differing client validation conflicts",
			ssls: []*adctypes.SSL{
				ssl(caOne, []adctypes.Certificate{certA}, "example.com"),
				ssl(caTwo, []adctypes.Certificate{certA}, "example.com"),
			},
			wantErr: `disagree on the client certificate validation of SNI "example.com"`,
		},
		{
			name: "adding client validation to an unauthenticated SNI conflicts",
			ssls: []*adctypes.SSL{
				ssl(nil, []adctypes.Certificate{certA}, "example.com"),
				ssl(caOne, []adctypes.Certificate{certA}, "example.com"),
			},
			wantErr: `disagree on the client certificate validation of SNI "example.com"`,
		},
		{
			name: "disjoint SNIs are left alone",
			ssls: []*adctypes.SSL{
				ssl(nil, []adctypes.Certificate{certA}, "a.example.com"),
				ssl(caOne, []adctypes.Certificate{certB}, "b.example.com"),
			},
			wantSnis: [][]string{{"a.example.com"}, {"b.example.com"}},
			wantCert: [][]adctypes.Certificate{{certA}, {certB}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dedupGatewaySSLSNIs(tt.ssls)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tt.wantSnis))
			for i, ssl := range got {
				assert.Equal(t, tt.wantSnis[i], ssl.Snis)
				assert.Equal(t, tt.wantCert[i], ssl.Certificates)
			}
		})
	}
}

// TestTranslateGateway_SameSNIAcrossListeners walks the two ends of the SNI
// collision through the whole translation. Two listeners pinning the same hostname
// on different ports are legal Gateway API, but the data plane keys SSL objects by
// SNI alone, so they have to agree on what that SNI is served with.
func TestTranslateGateway_SameSNIAcrossListeners(t *testing.T) {
	newContext := func() *provider.TranslateContext {
		tctx := provider.NewDefaultTranslateContext(context.Background())
		for _, name := range []string{"cert-one", "cert-two"} {
			tctx.Secrets[types.NamespacedName{Namespace: "default", Name: name}] = &corev1.Secret{
				Data: map[string][]byte{
					"cert": []byte(name + "-data"),
					"key":  []byte(name + "-key"),
				},
			}
		}
		tctx.ConfigMaps[types.NamespacedName{Namespace: "default", Name: "ca-cm"}] = &corev1.ConfigMap{
			Data: map[string]string{corev1.ServiceAccountRootCAKey: testCACert},
		}
		return tctx
	}
	newGateway := func(secondSecret string) *gatewayv1.Gateway {
		listener := func(name, secret string, port gatewayv1.PortNumber) gatewayv1.Listener {
			return gatewayv1.Listener{
				Name:     gatewayv1.SectionName(name),
				Port:     port,
				Protocol: gatewayv1.HTTPSProtocolType,
				Hostname: ptr.To(gatewayv1.Hostname("example.com")),
				TLS: &gatewayv1.ListenerTLSConfig{
					Mode: ptr.To(gatewayv1.TLSModeTerminate),
					CertificateRefs: []gatewayv1.SecretObjectReference{{
						Kind: ptr.To(gatewayv1.Kind("Secret")),
						Name: gatewayv1.ObjectName(secret),
					}},
				},
			}
		}
		return &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
			Spec: gatewayv1.GatewaySpec{
				Listeners: []gatewayv1.Listener{
					listener("https", "cert-one", 443),
					listener("https-alt", secondSecret, 8443),
				},
			},
		}
	}

	// Both listeners serve the same hostname with different certificates and no
	// client validation: the certificates are merged onto one SSL object rather
	// than one of them being dropped.
	t.Run("differing certificates are merged", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		result, err := tr.TranslateGateway(newContext(), newGateway("cert-two"))
		require.NoError(t, err)
		require.Len(t, result.SSL, 1)
		assert.Equal(t, []string{"example.com"}, result.SSL[0].Snis)
		assert.Equal(t, []adctypes.Certificate{
			{Certificate: "cert-one-data", Key: "cert-one-key"},
			{Certificate: "cert-two-data", Key: "cert-two-key"},
		}, result.SSL[0].Certificates)
	})

	// A per-port frontendValidation makes the two listeners disagree on which
	// clients may connect with that SNI. Only one policy can be programmed, so the
	// translation fails and the Gateway reports Accepted=False instead of silently
	// applying whichever listener came first.
	t.Run("differing client validation conflicts", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newGateway("cert-one")
		gateway.Spec.TLS = &gatewayv1.GatewayTLSConfig{
			Frontend: &gatewayv1.FrontendTLSConfig{
				PerPort: []gatewayv1.TLSPortConfig{{
					Port: 8443,
					TLS: gatewayv1.TLSConfig{
						Validation: &gatewayv1.FrontendTLSValidation{
							CACertificateRefs: []gatewayv1.ObjectReference{
								{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
							},
						},
					},
				}},
			},
		}

		_, err := tr.TranslateGateway(newContext(), gateway)
		require.ErrorContains(t, err, `disagree on the client certificate validation of SNI "example.com"`)
	})
}

func TestTranslateSecret_Passthrough(t *testing.T) {
	// A TLS passthrough listener does not terminate TLS, so it carries no
	// certificateRefs; translating it must not error, otherwise the whole Gateway
	// is rejected with Accepted=False.
	tr := &Translator{Log: logr.Discard()}
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls-passthrough",
					Protocol: gatewayv1.TLSProtocolType,
					Port:     443,
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayv1.TLSModePassthrough),
					},
				},
			},
		},
	}

	sslObjs, err := tr.translateSecret(newTranslateContextWithTLS(), gateway.Spec.Listeners[0], gateway)
	require.NoError(t, err)
	assert.Empty(t, sslObjs, "passthrough listener should not produce SSL objects")
}

func TestTranslateSecret_FrontendValidation(t *testing.T) {
	t.Run("with frontendValidation sets downstream mTLS client CA", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{
					Group: "",
					Kind:  "ConfigMap",
					Name:  "ca-cm",
				},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		require.NotNil(t, sslObjs[0].Client, "client mTLS config should be set")
		assert.Equal(t, testCACert, sslObjs[0].Client.CA)
		assert.Equal(t, []string{"example.com"}, sslObjs[0].Snis)
	})

	t.Run("with Secret CA ref sets downstream mTLS client CA", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "Secret", Name: "ca-secret"},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		require.NotNil(t, sslObjs[0].Client, "client mTLS config should be set")
		assert.Equal(t, testCACert, sslObjs[0].Client.CA)
	})

	t.Run("missing CA Secret returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "Secret", Name: "missing"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("without frontendValidation leaves client nil", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(nil)
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		assert.Nil(t, sslObjs[0].Client)
	})

	t.Run("AllowInsecureFallback emits no SSL for that listener", func(t *testing.T) {
		// The mode cannot be expressed on APISIX, so the listener is reported
		// Accepted=False/UnsupportedValue and programmed with nothing. Serving TLS
		// without the requested client validation would be less safe than serving
		// nothing, and failing here would take down the whole Gateway translation.
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			Mode: gatewayv1.AllowInsecureFallback,
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		assert.Empty(t, sslObjs)
	})

	t.Run("frontendValidation is ignored on a non-HTTPS listener", func(t *testing.T) {
		// spec.tls.frontend applies only to HTTPS listeners in Gateway API v1.6, so a
		// TLS (Terminate) listener must not get downstream mTLS attached.
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		gateway.Spec.Listeners[0].Protocol = gatewayv1.TLSProtocolType
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		assert.Nil(t, sslObjs[0].Client, "client mTLS must not be set for a non-HTTPS listener")
	})

	t.Run("missing CA ConfigMap returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "ConfigMap", Name: "missing"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("unsupported CA ref kind returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "Pod", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("unsupported CA ref group returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "example.com", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("malformed CA data returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()
		tctx.ConfigMaps[types.NamespacedName{Namespace: "default", Name: "ca-cm"}] = &corev1.ConfigMap{
			Data: map[string]string{corev1.ServiceAccountRootCAKey: "   not a pem cert   "},
		}

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})
}
