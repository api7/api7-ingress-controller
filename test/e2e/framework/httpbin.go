package framework

import (
	_ "embed"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/gomega"
)

var (
	//go:embed manifests/httpbin.yaml
	_httpbinSpec string
)

func (f *Framework) deployHTTPBIN() {
	f.GinkgoT.Log("deploying httpbin")
	err := k8s.KubectlApplyFromStringE(f.GinkgoT, f.kubectlOpts, _httpbinSpec)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "applying httpbin spec")

	err = f.ensureService("httpbin", _namespace, 2)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring httpbin service")
}
