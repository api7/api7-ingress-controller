package framework

import (
	_ "embed"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed manifests/mock-s3.yaml
	_mockS3Spec string
)

func (f *Framework) deployMockS3() {
	f.GinkgoT.Log("deploying mock-s3")
	err := k8s.KubectlApplyFromStringE(f.GinkgoT, f.kubectlOpts, _mockS3Spec)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "applying mock-s3 spec")

	err = f.ensureService("mock-s3", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred(), "ensuring mock-s3 service")
}

func (f *Framework) getMockS3ServiceIP() string {
	svc, err := f.clientset.CoreV1().Services(_namespace).Get(f.Context, "mock-s3", metav1.GetOptions{})
	f.GomegaT.Expect(err).ShouldNot(gomega.HaveOccurred())
	return svc.Spec.ClusterIP
}
