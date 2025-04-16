package manager

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/api7/api7-ingress-controller/internal/controller"
	"github.com/api7/api7-ingress-controller/internal/controller/indexer"
	"github.com/api7/api7-ingress-controller/internal/provider"
)

// K8s
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// CustomResourceDefinition
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=pluginconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=gatewayproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=consumers,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=consumers/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=backendtrafficpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=backendtrafficpolicies/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=httproutepolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.apisix.io,resources=httproutepolicies/status,verbs=get;update

// GatewayAPI
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update

// Networking
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch

type Controller interface {
	SetupWithManager(mgr manager.Manager) error
}

func setupControllers(ctx context.Context, mgr manager.Manager, pro provider.Provider) ([]Controller, error) {
	if err := indexer.SetupIndexer(mgr); err != nil {
		return nil, err
	}
	return []Controller{
		&controller.GatewayClassReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Log:    ctrl.LoggerFrom(ctx).WithName("controllers").WithName("GatewayClass"),
		},
		&controller.GatewayReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Gateway"),
			Provider: pro,
		},
		&controller.HTTPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("HTTPRoute"),
			Provider: pro,
		},
		&controller.IngressReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Ingress"),
			Provider: pro,
		},
		&controller.ConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Consumer"),
			Provider: pro,
		},
	}, nil
}
