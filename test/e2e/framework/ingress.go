package framework

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	. "github.com/onsi/gomega" //nolint:staticcheck
	"github.com/pkg/errors"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed manifests/ingress.yaml
	_ingressSpec   string
	IngressSpecTpl *template.Template
)

func init() {
	tpl, err := template.New("ingress").Funcs(sprig.TxtFuncMap()).Parse(_ingressSpec)
	if err != nil {
		panic(err)
	}
	IngressSpecTpl = tpl
}

type IngressDeployOpts struct {
	ControllerName string
	AdminKey       string
	AdminTLSVerify bool
	Namespace      string
	AdminEnpoint   string
	StatusAddress  string
	TLSKey         string
	TLSCRT         string
	CaBundle       string
}

func (f *Framework) DeployIngress(opts IngressDeployOpts) {
	err := f.setWebhookCertificate(&opts)
	f.GomegaT.Expect(err).NotTo(HaveOccurred(), "set certificates info for webhook service")

	buf := bytes.NewBuffer(nil)

	err = IngressSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering ingress spec")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err = WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "waiting for controller-manager pod ready")
	f.WaitControllerManagerLog("All cache synced successfully", 0, time.Minute)
}

func (f *Framework) setWebhookCertificate(opts *IngressDeployOpts) error {
	// generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return errors.Wrap(err, "failed to GenerateKey")
	}
	var privateKeyBuf = bytes.NewBuffer(nil)
	if err = pem.Encode(privateKeyBuf, &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return errors.Wrap(err, "failed to pem.Encode private key")
	}

	// prepare CertificateSigningRequest
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{"system:nodes"},
			CommonName:   "system:node:webhook-service." + opts.Namespace,
		},
		DNSNames: []string{
			"webhook-service",
			"webhook-service." + opts.Namespace,
			"webhook-service." + opts.Namespace + ".svc",
		},
	}, privateKey)
	if err != nil {
		return errors.Wrap(err, "failed to CreateCertificateRequest")
	}

	certificateSigningRequest := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook-csr",
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request: pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csrBytes,
			}),
			SignerName:        "kubernetes.io/kubelet-serving",
			ExpirationSeconds: nil,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageServerAuth,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
			},
		},
	}

	// try to delete the CertificateSigningRequest before creating
	err = f.clientset.CertificatesV1().CertificateSigningRequests().
		Delete(context.Background(), certificateSigningRequest.GetName(), metav1.DeleteOptions{})
	if err != nil {
		f.Logf("failed to CertificateSigningRequests().Delete: %v", err)
	}

	// create CertificateSigningRequest
	certificateSigningRequest, err = f.clientset.CertificatesV1().CertificateSigningRequests().
		Create(context.Background(), certificateSigningRequest, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to CertificateSigningRequests().Create")
	}

	// to approve the CertificateSigningRequest
	condition := certificatesv1.CertificateSigningRequestCondition{
		Type:               certificatesv1.CertificateApproved,
		Status:             "True",
		Reason:             "AdminApproval",
		Message:            "CSR approved by admin",
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
	}
	certificateSigningRequest.Status.Conditions = append(certificateSigningRequest.Status.Conditions, condition)
	certificateSigningRequest, err = f.clientset.CertificatesV1().CertificateSigningRequests().
		UpdateApproval(context.Background(), certificateSigningRequest.GetName(), certificateSigningRequest, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to CertificateSigningRequests().UpdateApproval")
	}

	// try to get the Certificate issued by K8s
	certPEM := retry.DoWithRetry(f.GinkgoT, "get approved certificate", 10, time.Second, func() (string, error) {
		csr, err := f.clientset.CertificatesV1().CertificateSigningRequests().Get(context.Background(), certificateSigningRequest.GetName(), metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if csr.Status.Certificate == nil {
			return "", errors.New("certificate is not signed yet")
		}
		return string(csr.Status.Certificate), nil
	})

	// get client-ca-file as the caBundle for webhook ValidatingWebhookConfiguration
	var cm corev1.ConfigMap
	var cmKey = client.ObjectKey{
		Namespace: "kube-system",
		Name:      "extension-apiserver-authentication",
	}
	err = f.K8sClient.Get(context.Background(), cmKey, &cm)
	if err != nil {
		return errors.Wrapf(err, "failed to get ConfigMap: %v", cmKey)
	}

	// set certificate info
	opts.TLSKey = "\n" + privateKeyBuf.String()
	opts.TLSCRT = "\n" + certPEM
	opts.CaBundle = base64.StdEncoding.EncodeToString([]byte(cm.Data["client-ca-file"]))

	return nil
}
