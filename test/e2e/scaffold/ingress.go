package scaffold

import (
	"bytes"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Scaffold) deployIngress() {
	buf := bytes.NewBuffer(nil)

	err := framework.IngressSpecTpl.Execute(buf, map[string]any{
		"Namespace":      s.namespace,
		"AdminKey":       s.AdminKey(),
		"AdminEnpoint":   framework.DashboardTLSEndpoint + "/apisix/admin",
		"AdminTLSVerify": false,
		"ControllerName": s.opts.ControllerName,
	})
	Expect(err).ToNot(HaveOccurred(), "rendering ingress spec")

	k8s.KubectlApplyFromString(s.t, s.kubectlOptions, buf.String())

	err = s.waitPodsAvailable(metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	Expect(err).ToNot(HaveOccurred(), "waiting for controller-manager pod ready")
}
