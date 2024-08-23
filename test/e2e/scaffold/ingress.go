package scaffold

import (
	"bytes"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	"github.com/gruntwork-io/terratest/modules/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Scaffold) deployIngress() {
	buf := bytes.NewBuffer(nil)

	framework.IngressSpecTpl.Execute(buf, map[string]any{
		"Namespace":           s.namespace,
		"AdminKey":            s.AdminKey(),
		"ControlPlaneEnpoint": framework.DashboardEndpoint,
		"ControllerName":      "gateway.api7.io/api7-ingress-controller",
	})

	k8s.KubectlApplyFromString(s.t, s.kubectlOptions, buf.String())

	s.waitPodsAvailable(metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
}
