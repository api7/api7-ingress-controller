// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateApisixTls(t *testing.T) {
	translator := &Translator{}

	// Test basic TLS without mutual TLS
	t.Run("basic TLS", func(t *testing.T) {
		// Create test secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("test-cert"),
				"tls.key": []byte("test-key"),
			},
		}

		// Create test ApisixTls
		tls := &apiv2.ApisixTls{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls",
				Namespace: "test-ns",
			},
			Spec: apiv2.ApisixTlsSpec{
				Hosts: []apiv2.HostType{"example.com", "*.example.com"},
				Secret: apiv2.ApisixSecret{
					Name:      "test-secret",
					Namespace: "test-ns",
				},
			},
		}

		// Create translate context
		tctx := &provider.TranslateContext{
			Secrets: map[types.NamespacedName]*corev1.Secret{
				{Name: "test-secret", Namespace: "test-ns"}: secret,
			},
		}

		// Test translation
		result, err := translator.TranslateApisixTls(tctx, tls)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.SSL, 1)

		ssl := result.SSL[0]
		assert.NotEmpty(t, ssl.ID) // ID is generated hash, so just check it's not empty
		assert.Len(t, ssl.Certificates, 1)
		assert.Equal(t, "test-cert", ssl.Certificates[0].Certificate)
		assert.Equal(t, "test-key", ssl.Certificates[0].Key)
		assert.Equal(t, []string{"example.com", "*.example.com"}, ssl.Snis)
		assert.Nil(t, ssl.Client)
	})

	// Test TLS with mutual TLS
	t.Run("TLS with mutual TLS", func(t *testing.T) {
		// Create test secrets
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("test-cert"),
				"tls.key": []byte("test-key"),
			},
		}

		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ca-secret",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"ca.crt": []byte("test-ca"),
			},
		}

		// Create test ApisixTls
		tls := &apiv2.ApisixTls{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls",
				Namespace: "test-ns",
			},
			Spec: apiv2.ApisixTlsSpec{
				Hosts: []apiv2.HostType{"example.com"},
				Secret: apiv2.ApisixSecret{
					Name:      "test-secret",
					Namespace: "test-ns",
				},
				Client: &apiv2.ApisixMutualTlsClientConfig{
					CASecret: apiv2.ApisixSecret{
						Name:      "ca-secret",
						Namespace: "test-ns",
					},
					Depth:            2,
					SkipMTLSUriRegex: []string{"/health", "/metrics"},
				},
			},
		}

		// Create translate context
		tctx := &provider.TranslateContext{
			Secrets: map[types.NamespacedName]*corev1.Secret{
				{Name: "test-secret", Namespace: "test-ns"}: secret,
				{Name: "ca-secret", Namespace: "test-ns"}:   caSecret,
			},
		}

		// Test translation
		result, err := translator.TranslateApisixTls(tctx, tls)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.SSL, 1)

		ssl := result.SSL[0]
		assert.NotEmpty(t, ssl.ID) // ID is generated hash, so just check it's not empty
		assert.Len(t, ssl.Certificates, 1)
		assert.Equal(t, "test-cert", ssl.Certificates[0].Certificate)
		assert.Equal(t, "test-key", ssl.Certificates[0].Key)
		assert.Equal(t, []string{"example.com"}, ssl.Snis)
		assert.NotNil(t, ssl.Client)
		assert.Equal(t, "test-ca", ssl.Client.CA)
		assert.Equal(t, int64(2), *ssl.Client.Depth)
		assert.Equal(t, []string{"/health", "/metrics"}, ssl.Client.SkipMtlsURIRegex)
	})

	// Test with missing secret
	t.Run("missing secret", func(t *testing.T) {
		// Create test ApisixTls
		tls := &apiv2.ApisixTls{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls",
				Namespace: "test-ns",
			},
			Spec: apiv2.ApisixTlsSpec{
				Hosts: []apiv2.HostType{"example.com"},
				Secret: apiv2.ApisixSecret{
					Name:      "missing-secret",
					Namespace: "test-ns",
				},
			},
		}

		// Create empty translate context
		tctx := &provider.TranslateContext{
			Secrets: map[types.NamespacedName]*corev1.Secret{},
		}

		// Test translation
		result, err := translator.TranslateApisixTls(tctx, tls)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.SSL, 0) // Should return empty result when secret is missing
	})
}
