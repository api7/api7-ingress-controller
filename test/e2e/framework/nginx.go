package framework

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed manifests/nginx.yaml
	_ngxSpec   string
	ngxSpecTpl *template.Template
)

type NginxOptions struct {
	Namespace string
}

func init() {
	tpl, err := template.New("ngx").Funcs(sprig.TxtFuncMap()).Parse(_ngxSpec)
	if err != nil {
		panic(err)
	}
	ngxSpecTpl = tpl
}

func (f *Framework) DeployNginx(opts NginxOptions) *corev1.Service {
	buf := bytes.NewBuffer(nil)

	err := ngxSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering nginx spec")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "app=nginx",
	})

	return k8s.GetService(f.GinkgoT, kubectlOpts, "nginx")
}
