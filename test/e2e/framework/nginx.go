package framework

import (
	_ "embed"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/gomega"
)

var (
	//go:embed manifests/nginx.yaml
	_nginxSpec string
)

func (f *Framework) deployNginx() {
	f.applySSLSecret("nginx-ssl", []byte(TESTCert1), []byte(TestKey1), []byte(TestCACert))

	f.GinkgoT.Log("deploying nginx")
	err := k8s.KubectlApplyFromStringE(f.GinkgoT, f.kubectlOpts, _nginxSpec)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "applying nginx spec")

	err = f.ensureService("nginx", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring nginx service")
}
