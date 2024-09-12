package controlplane

import (
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controlplane/translator"
	"github.com/api7/api7-ingress-controller/pkg/dashboard"
	"github.com/api7/gopkg/pkg/log"
)

type Controlplane interface {
	Update(context.Context, *translator.TranslateContext, client.Object) error
	Delete(context.Context, client.Object) error
}

type dashboardClient struct {
	translator *translator.Translator
	c          dashboard.Dashboard
}

func NewDashboard() (Controlplane, error) {
	control, err := dashboard.NewClient()
	if err != nil {
		return nil, err
	}

	gc := config.GetFirstGatewayConfig()
	if err := control.AddCluster(context.TODO(), &dashboard.ClusterOptions{
		Name:          "default",
		BaseURL:       gc.ControlPlane.Endpoints[0],
		AdminKey:      gc.ControlPlane.AdminKey,
		SkipTLSVerify: !*gc.ControlPlane.TLSVerify,
	}); err != nil {
		return nil, err
	}

	return &dashboardClient{
		translator: &translator.Translator{
			Log: ctrl.Log.WithName("controlplane").WithName("translator"),
		},
		c: control,
	}, nil
}

func (d *dashboardClient) Update(ctx context.Context, tctx *translator.TranslateContext, obj client.Object) error {
	var result *translator.TranslateResult
	var err error
	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateGatewayHTTPRoute(tctx, obj.DeepCopy())
	}
	if err != nil {
		return err
	}

	clusterName := "default"

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	namespace := obj.GetNamespace()
	name := obj.GetName()

	KindLabel := dashboard.ListByKindLabelOptions{
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
	}

	routes, err := d.c.Cluster(clusterName).Route().List(ctx, dashboard.ListOptions{
		From:      dashboard.ListFromCache,
		KindLabel: KindLabel,
	})
	log.Infow("route", zap.Any("route", routes))
	if err != nil && err != dashboard.ErrNotFound {
		return err
	}

	service, err := d.c.Cluster(clusterName).Service().List(ctx, dashboard.ListOptions{
		From:      dashboard.ListFromCache,
		KindLabel: KindLabel,
	})
	log.Infow("service", zap.Any("service", service))
	if err != nil && err != dashboard.ErrNotFound {
		return err
	}

	// Delete the routes and services that are not in the result
	for _, route := range routes {
		if _, ok := result.RouteMap[route.Name]; ok {
			continue
		}
		if err := d.c.Cluster(clusterName).Route().Delete(ctx, route); err != nil {
			return err
		}
	}

	for _, service := range service {
		if _, ok := result.ServiceMap[service.Name]; ok {
			continue
		}
		if err := d.c.Cluster(clusterName).Service().Delete(ctx, service); err != nil {
			return err
		}
	}

	for _, service := range result.ServiceMap {
		if _, err := d.c.Cluster(clusterName).Service().Update(ctx, service); err != nil {
			return err
		}
	}
	for _, route := range result.RouteMap {
		if _, err := d.c.Cluster(clusterName).Route().Update(ctx, route); err != nil {
			return err
		}
	}
	return nil
}

func (d *dashboardClient) Delete(ctx context.Context, obj client.Object) error {
	clusters := d.c.ListClusters()
	kindLabel := dashboard.ListByKindLabelOptions{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	for _, cluster := range clusters {
		routes, _ := cluster.Route().List(ctx, dashboard.ListOptions{
			From:      dashboard.ListFromCache,
			KindLabel: kindLabel,
		})

		for _, route := range routes {
			if err := cluster.Route().Delete(ctx, route); err != nil {
				return err
			}
		}

		services, _ := cluster.Service().List(ctx, dashboard.ListOptions{
			From:      dashboard.ListFromCache,
			KindLabel: kindLabel,
		})

		for _, service := range services {
			if err := cluster.Service().Delete(ctx, service); err != nil {
				return err
			}
		}
	}
	return nil
}
