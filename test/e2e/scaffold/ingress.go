package scaffold

import (
	"github.com/api7/api7-ingress-controller/test/e2e/framework"
)

func (s *Scaffold) deployIngress() {
	s.DeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		AdminKey:       s.AdminKey(),
		AdminTLSVerify: false,
		Namespace:      s.namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint,
	})
}
