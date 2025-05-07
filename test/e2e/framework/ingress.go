package framework

import (
	"bytes"
	_ "embed"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega" //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Replicas       int
}

func (f *Framework) DeployIngress(opts IngressDeployOpts) {
	buf := bytes.NewBuffer(nil)

	err := IngressSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering ingress spec")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err = WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "waiting for controller-manager pod ready")
	f.WaitControllerManagerLog("All cache synced successfully", 0, time.Minute)
}
