package controlplane

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/api7/api7-ingress-controller/internal/controller/config"
	"github.com/api7/api7-ingress-controller/internal/controlplane/translator"
	"github.com/api7/api7-ingress-controller/pkg/dashboard"
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
		translator: &translator.Translator{},
		c:          control,
	}, nil
}

func (d *dashboardClient) Update(ctx context.Context, tctx *translator.TranslateContext, obj client.Object) error {
	var result *translator.TranslateResult
	var err error
	switch obj := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateGatewayHTTPRoute(tctx, obj.DeepCopy())
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, obj.DeepCopy())
	}
	if err != nil {
		return err
	}
	// TODO: support diff resources
	name := "default"
	for _, service := range result.Services {
		if _, err := d.c.Cluster(name).Service().Update(ctx, service); err != nil {
			return err
		}
	}
	for _, route := range result.Routes {
		if _, err := d.c.Cluster(name).Route().Update(ctx, route); err != nil {
			return err
		}
	}
	for _, ssl := range result.SSL {
		if _, err := d.c.Cluster(name).SSL().Update(ctx, ssl); err != nil {
			return err
		}
	}
	return nil
}

func (d *dashboardClient) Delete(ctx context.Context, obj client.Object) error {
	clusters := d.c.ListClusters()
	for _, cluster := range clusters {
		routes, _ := cluster.Route().List(ctx, dashboard.ListOptions{
			From: dashboard.ListFromCache,
			Args: []interface{}{
				"label",
				obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetNamespace(),
				obj.GetName(),
			},
		})

		for _, route := range routes {
			if err := cluster.Route().Delete(ctx, route); err != nil {
				return err
			}
		}

		services, _ := cluster.Service().List(ctx, dashboard.ListOptions{
			From: dashboard.ListFromCache,
			Args: []interface{}{
				"label",
				obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetNamespace(),
				obj.GetName(),
			},
		})

		for _, service := range services {
			if err := cluster.Service().Delete(ctx, service); err != nil {
				return err
			}
		}

		ssls, _ := cluster.SSL().List(ctx)
		for _, ssl := range ssls {
			if err := cluster.SSL().Delete(ctx, ssl); err != nil {
				return err
			}
		}
	}
	return nil
}
