package manager

import (
	"context"

	"github.com/api7/api7-ingress-controller/internal/controller"
	"github.com/api7/api7-ingress-controller/internal/controlplane"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

type Controller interface {
	SetupWithManager(mgr manager.Manager) error
}

func setupControllers(ctx context.Context, mgr manager.Manager, cpclient controlplane.Controlplane) []Controller {

	return []Controller{
		&controller.GatewayClassReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Log:    ctrl.LoggerFrom(ctx).WithName("controllers").WithName("GatewayClass"),
		},
		&controller.GatewayReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Log:    ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Gateway"),
		},
		&controller.HTTPRouteReconciler{
			Client:             mgr.GetClient(),
			Scheme:             mgr.GetScheme(),
			Log:                ctrl.LoggerFrom(ctx).WithName("controllers").WithName("HTTPRoute"),
			ControlPalneClient: cpclient,
		},
	}
}
