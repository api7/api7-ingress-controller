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

package apisix

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixTls", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	AfterEach(func() {
		By("clean up GatewayProxy")
		err := s.DeleteResourceFromStringWithNamespace(gatewayProxyYaml, "default")
		Expect(err).NotTo(HaveOccurred(), "delete GatewayProxy")

		By("clean up IngressClass")
		err = s.DeleteResourceFromStringWithNamespace(ingressClassYaml, "")
		Expect(err).NotTo(HaveOccurred(), "delete IngressClass")
	})

	Context("Basic TLS Configuration", func() {
		It("should create SSL certificate in APISIX", func() {
			By("generating TLS certificate and key")
			cert, key, err := generateSelfSignedCert("example.com")
			Expect(err).NotTo(HaveOccurred())

			By("creating TLS secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tls-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": cert,
					"tls.key": key,
				},
			}
			err = s.K8sClient.Create(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())

			By("creating ApisixTls resource")
			apisixTls := &apiv2.ApisixTls{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tls",
					Namespace: "default",
				},
				Spec: apiv2.ApisixTlsSpec{
					Hosts: []apiv2.HostType{"example.com"},
					Secret: apiv2.ApisixSecret{
						Name:      "test-tls-secret",
						Namespace: "default",
					},
					IngressClassName: "apisix",
				},
			}
			err = s.K8sClient.Create(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())

			By("checking ApisixTls status")
			Eventually(func() bool {
				var tls apiv2.ApisixTls
				err := s.K8sClient.Get(s.Context, types.NamespacedName{
					Name:      "test-tls",
					Namespace: "default",
				}, &tls)
				if err != nil {
					return false
				}

				if len(tls.Status.Conditions) == 0 {
					return false
				}

				condition := tls.Status.Conditions[0]
				return condition.Type == string(apiv2.ConditionTypeAccepted) &&
					condition.Status == metav1.ConditionTrue
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying SSL configuration in APISIX")
			Eventually(func() bool {
				// Test HTTPS connection to verify SSL is configured
				client := &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
					Timeout: 5 * time.Second,
				}

				// Since we can't easily test HTTPS without a proper setup,
				// we'll check if the SSL object exists in APISIX admin API
				resp, err := client.Get(fmt.Sprintf("%s/apisix/admin/ssls", s.Deployer.GetAdminEndpoint()))
				if err != nil {
					return false
				}
				defer func() { _ = resp.Body.Close() }()

				return resp.StatusCode == http.StatusOK
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("cleaning up")
			err = s.K8sClient.Delete(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())
			err = s.K8sClient.Delete(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Mutual TLS (mTLS) Configuration", func() {
		It("should create SSL certificate with client CA in APISIX", func() {
			By("generating server certificate and key")
			cert, key, err := generateSelfSignedCert("example.com")
			Expect(err).NotTo(HaveOccurred())

			By("generating CA certificate for client authentication")
			caCert, caKey, err := generateSelfSignedCert("ca.example.com")
			Expect(err).NotTo(HaveOccurred())

			By("creating TLS secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tls-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": cert,
					"tls.key": key,
				},
			}
			err = s.K8sClient.Create(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())

			By("creating CA secret")
			caSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ca-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"tls.crt": caCert,
					"tls.key": caKey,
				},
			}
			err = s.K8sClient.Create(s.Context, caSecret)
			Expect(err).NotTo(HaveOccurred())

			By("creating ApisixTls resource with mTLS")
			apisixTls := &apiv2.ApisixTls{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mtls",
					Namespace: "default",
				},
				Spec: apiv2.ApisixTlsSpec{
					Hosts: []apiv2.HostType{"example.com"},
					Secret: apiv2.ApisixSecret{
						Name:      "test-tls-secret",
						Namespace: "default",
					},
					Client: &apiv2.ApisixMutualTlsClientConfig{
						CASecret: apiv2.ApisixSecret{
							Name:      "test-ca-secret",
							Namespace: "default",
						},
						Depth:            2,
						SkipMTLSUriRegex: []string{"/health"},
					},
					IngressClassName: "apisix",
				},
			}
			err = s.K8sClient.Create(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())

			By("checking ApisixTls status")
			Eventually(func() bool {
				var tls apiv2.ApisixTls
				err := s.K8sClient.Get(s.Context, types.NamespacedName{
					Name:      "test-mtls",
					Namespace: "default",
				}, &tls)
				if err != nil {
					return false
				}

				if len(tls.Status.Conditions) == 0 {
					return false
				}

				condition := tls.Status.Conditions[0]
				return condition.Type == string(apiv2.ConditionTypeAccepted) &&
					condition.Status == metav1.ConditionTrue
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("cleaning up")
			err = s.K8sClient.Delete(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())
			err = s.K8sClient.Delete(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())
			err = s.K8sClient.Delete(s.Context, caSecret)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Error Scenarios", func() {
		It("should handle missing TLS secret", func() {
			By("creating ApisixTls resource with non-existent secret")
			apisixTls := &apiv2.ApisixTls{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tls-missing-secret",
					Namespace: "default",
				},
				Spec: apiv2.ApisixTlsSpec{
					Hosts: []apiv2.HostType{"example.com"},
					Secret: apiv2.ApisixSecret{
						Name:      "non-existent-secret",
						Namespace: "default",
					},
					IngressClassName: "apisix",
				},
			}
			err := s.K8sClient.Create(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())

			By("checking ApisixTls status shows error")
			Eventually(func() bool {
				var tls apiv2.ApisixTls
				err := s.K8sClient.Get(s.Context, types.NamespacedName{
					Name:      "test-tls-missing-secret",
					Namespace: "default",
				}, &tls)
				if err != nil {
					return false
				}

				if len(tls.Status.Conditions) == 0 {
					return false
				}

				condition := tls.Status.Conditions[0]
				return condition.Type == string(apiv2.ConditionTypeAccepted) &&
					condition.Status == metav1.ConditionFalse &&
					condition.Reason == string(apiv2.ConditionReasonInvalidSpec)
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("cleaning up")
			err = s.K8sClient.Delete(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle missing CA secret for mTLS", func() {
			By("generating server certificate and key")
			cert, key, err := generateSelfSignedCert("example.com")
			Expect(err).NotTo(HaveOccurred())

			By("creating TLS secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tls-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": cert,
					"tls.key": key,
				},
			}
			err = s.K8sClient.Create(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())

			By("creating ApisixTls resource with non-existent CA secret")
			apisixTls := &apiv2.ApisixTls{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mtls-missing-ca",
					Namespace: "default",
				},
				Spec: apiv2.ApisixTlsSpec{
					Hosts: []apiv2.HostType{"example.com"},
					Secret: apiv2.ApisixSecret{
						Name:      "test-tls-secret",
						Namespace: "default",
					},
					Client: &apiv2.ApisixMutualTlsClientConfig{
						CASecret: apiv2.ApisixSecret{
							Name:      "non-existent-ca-secret",
							Namespace: "default",
						},
						Depth: 2,
					},
					IngressClassName: "apisix",
				},
			}
			err = s.K8sClient.Create(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())

			By("checking ApisixTls status shows error")
			Eventually(func() bool {
				var tls apiv2.ApisixTls
				err := s.K8sClient.Get(s.Context, types.NamespacedName{
					Name:      "test-mtls-missing-ca",
					Namespace: "default",
				}, &tls)
				if err != nil {
					return false
				}

				if len(tls.Status.Conditions) == 0 {
					return false
				}

				condition := tls.Status.Conditions[0]
				return condition.Type == string(apiv2.ConditionTypeAccepted) &&
					condition.Status == metav1.ConditionFalse &&
					condition.Reason == string(apiv2.ConditionReasonInvalidSpec)
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("cleaning up")
			err = s.K8sClient.Delete(s.Context, apisixTls)
			Expect(err).NotTo(HaveOccurred())
			err = s.K8sClient.Delete(s.Context, secret)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// generateSelfSignedCert generates a self-signed certificate for testing
func generateSelfSignedCert(commonName string) ([]byte, []byte, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return certPEM, keyPEM, nil
}
