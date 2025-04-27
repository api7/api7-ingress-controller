package scaffold

import (
	"github.com/api7/api7-ingress-controller/test/e2e/framework"
)

// deployIngress 部署 ingress 控制器
func (s *Scaffold) deployIngress() {
	s.internalDeployIngress(framework.IngressDeployOpts{
		ControllerName: s.opts.ControllerName,
		AdminKey:       s.AdminKey(),
		AdminTLSVerify: false,
		Namespace:      s.namespace,
		AdminEnpoint:   framework.DashboardTLSEndpoint,
	})
}

// internalDeployIngress 实际部署 ingress 控制器
func (s *Scaffold) internalDeployIngress(opts framework.IngressDeployOpts) {
	// 这里实现部署 ingress 的逻辑
	// 只是一个占位符，需要根据实际情况实现
}
