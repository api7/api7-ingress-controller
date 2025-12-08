// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package manager

import (
	"context"

	"github.com/go-logr/logr"
	netv1 "k8s.io/api/networking/v1"
	netv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	types "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/pkg/utils"
<<<<<<< HEAD
=======
	"github.com/go-logr/logr"
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
)

// K8s
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch

// CustomResourceDefinition v2
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixconsumers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixpluginconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixtlses,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixupstreams,verbs=get;list;watch

// CustomResourceDefinition v2 status
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixconsumers/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixpluginconfigs/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixroutes/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixtlses/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixupstreams/status,verbs=get;update

// CustomResourceDefinition
// +kubebuilder:rbac:groups=apisix.apache.org,resources=pluginconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=gatewayproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=consumers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=consumers/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=backendtrafficpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=backendtrafficpolicies/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=httproutepolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=httproutepolicies/status,verbs=get;update

// GatewayAPI
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=grpcroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=grpcroutes/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tlsroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tlsroutes/status,verbs=get;update

// Networking
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch

type Controller interface {
	SetupWithManager(mgr manager.Manager) error
}

func setupControllers(ctx context.Context, mgr manager.Manager, pro provider.Provider, updater status.Updater, readier readiness.ReadinessManager) ([]Controller, error) {
	setupLog := ctrl.LoggerFrom(ctx).WithName("setup")

	if err := indexer.SetupAPIv1alpha1Indexer(mgr); err != nil {
		setupLog.Error(err, "failed to setup v1alpha1 indexer")
		return nil, err
	}

<<<<<<< HEAD
	setupLog := ctrl.LoggerFrom(ctx).WithName("setup")
	var controllers []Controller

	icgv := netv1.SchemeGroupVersion
	if !utils.HasAPIResource(mgr, &netv1.IngressClass{}) {
		setupLog.Info("IngressClass v1 not found, falling back to IngressClass v1beta1")
		icgv = netv1beta1.SchemeGroupVersion
		controllers = append(controllers, &controller.IngressClassV1beta1Reconciler{
=======
	runnables := []Controller{}
	if controllers, err := setupGatewayAPIControllers(ctx, mgr, pro, updater, readier); err != nil {
		setupLog.Error(err, "failed to setup Gateway API controllers")
		return nil, err
	} else {
		runnables = append(runnables, controllers...)
	}

	if controllers, err := setupAPIv2Controllers(ctx, mgr, pro, updater, readier); err != nil {
		setupLog.Error(err, "failed to setup API v2 controllers")
		return nil, err
	} else {
		runnables = append(runnables, controllers...)
	}

	// required controller
	runnables = append(runnables, &controller.GatewayProxyController{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGatewayProxy),
		Provider: pro,
	})
	return runnables, nil
}

func setupGatewayAPIControllers(ctx context.Context, mgr manager.Manager, pro provider.Provider, updater status.Updater, readier readiness.ReadinessManager) ([]Controller, error) {
	if err := indexer.SetupGatewayAPIIndexer(mgr); err != nil {
		return nil, err
	}

	setupLog := ctrl.LoggerFrom(ctx).WithName("setup").WithName("gatewayapi")
	runnables := []Controller{}
	for resource, controller := range map[client.Object]Controller{
		&gatewayv1.GatewayClass{}: &controller.GatewayClassReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGatewayClass),
			Updater: updater,
		},
		&gatewayv1.Gateway{}: &controller.GatewayReconciler{
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindIngressClass),
			Provider: pro,
<<<<<<< HEAD
		})
	}

	// Gateway API Controllers - conditional registration based on API availability
	if !config.ControllerConfig.DisableGatewayAPI {
		for resource, controller := range map[client.Object]Controller{
			&gatewayv1.GatewayClass{}: &controller.GatewayClassReconciler{
				Client:  mgr.GetClient(),
				Scheme:  mgr.GetScheme(),
				Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGatewayClass),
				Updater: updater,
			},
			&gatewayv1.Gateway{}: &controller.GatewayReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGateway),
				Provider: pro,
				Updater:  updater,
			},
			&gatewayv1.HTTPRoute{}: &controller.HTTPRouteReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindHTTPRoute),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
			&gatewayv1alpha2.TCPRoute{}: &controller.TCPRouteReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindTCPRoute),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
			&gatewayv1alpha2.UDPRoute{}: &controller.UDPRouteReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindUDPRoute),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
			&gatewayv1.GRPCRoute{}: &controller.GRPCRouteReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGRPCRoute),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
			&gatewayv1alpha2.TLSRoute{}: &controller.TLSRouteReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindTLSRoute),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
			&v1alpha1.Consumer{}: &controller.ConsumerReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindConsumer),
				Provider: pro,
				Updater:  updater,
				Readier:  readier,
			},
		} {
			if utils.HasAPIResource(mgr, resource) {
				controllers = append(controllers, controller)
			} else {
				setupLog.Info("Skipping controller setup, API not found in cluster", "api", utils.FormatGVK(resource))
			}
		}
	} else {
		setupLog.Info("Skipping Gateway API controllers setup as Gateway API is disabled")
	}

	for resource, controller := range map[client.Object]Controller{
		&netv1.Ingress{}: &controller.IngressReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindIngress),
=======
			Updater:  updater,
		},
		&gatewayv1.HTTPRoute{}: &controller.HTTPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindHTTPRoute),
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
<<<<<<< HEAD
=======
		&gatewayv1.GRPCRoute{}: &controller.GRPCRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGRPCRoute),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&gatewayv1alpha2.TCPRoute{}: &controller.TCPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindTCPRoute),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&gatewayv1alpha2.UDPRoute{}: &controller.UDPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindUDPRoute),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&gatewayv1alpha2.TLSRoute{}: &controller.TLSRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindTLSRoute),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&v1alpha1.Consumer{}: &controller.ConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindConsumer),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
	} {
		if utils.HasAPIResource(mgr, resource) {
			runnables = append(runnables, controller)
		} else {
			setupLog.Info("Skipping indexer setup, API not found in cluster", "api", utils.FormatGVK(resource))
		}
	}
	return runnables, nil
}

func setupAPIv2Controllers(ctx context.Context, mgr manager.Manager, pro provider.Provider, updater status.Updater, readier readiness.ReadinessManager) ([]Controller, error) {
	if err := indexer.SetupAPIv2Indexer(mgr); err != nil {
		return nil, err
	}

	setupLog := ctrl.LoggerFrom(ctx).WithName("setup").WithName("apiv2")

	runnables := []Controller{}
	for resource, controller := range map[client.Object]Controller{
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
		&netv1.IngressClass{}: &controller.IngressClassReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindIngressClass),
			Provider: pro,
		},
<<<<<<< HEAD
	} {
		if utils.HasAPIResource(mgr, resource) {
			controllers = append(controllers, controller)
		} else {
			setupLog.Info("Skipping controller setup, API not found in cluster", "api", utils.FormatGVK(resource))
		}
	}

	controllers = append(controllers, []Controller{
		// Gateway Proxy Controller - always register this as it is core to the controller
		&controller.GatewayProxyController{
=======
		&netv1.Ingress{}: &controller.IngressReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindIngress),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&apiv2.ApisixGlobalRule{}: &controller.ApisixGlobalRuleReconciler{
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindGatewayProxy),
			Provider: pro,
			ICGV:     icgv,
		},
		&apiv2.ApisixRoute{}: &controller.ApisixRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixRoute),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGV:     icgv,
		},
		&controller.ApisixGlobalRuleReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixGlobalRule),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGV:     icgv,
		},
		&apiv2.ApisixConsumer{}: &controller.ApisixConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixConsumer),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGV:     icgv,
		},
		&apiv2.ApisixPluginConfig{}: &controller.ApisixPluginConfigReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixPluginConfig),
			Updater: updater,
			ICGV:    icgv,
		},
		&apiv2.ApisixTls{}: &controller.ApisixTlsReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixTls),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGV:     icgv,
		},
		&apiv2.ApisixUpstream{}: &controller.ApisixUpstreamReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName(types.KindApisixUpstream),
			Updater: updater,
<<<<<<< HEAD
			ICGV:    icgv,
		},
	}...)

	setupLog.Info("Controllers setup completed", "total_controllers", len(controllers))
	return controllers, nil
}

func registerReadinessGVK(mgr manager.Manager, readier readiness.ReadinessManager) {
	log := ctrl.LoggerFrom(context.Background()).WithName("readiness")

	registerV2ForReadinessGVK(mgr, readier, log)
	if !config.ControllerConfig.DisableGatewayAPI {
		registerGatewayAPIForReadinessGVK(mgr, readier, log)
		registerV1alpha1ForReadinessGVK(mgr, readier, log)
	} else {
		log.Info("Skipping Gateway API and v1alpha1 GVK registration for readiness checks as Gateway API is disabled")
	}
}

func registerV2ForReadinessGVK(mgr manager.Manager, readier readiness.ReadinessManager, log logr.Logger) {
	icgv := netv1.SchemeGroupVersion
	if !utils.HasAPIResource(mgr, &netv1.IngressClass{}) {
		icgv = netv1beta1.SchemeGroupVersion
	}

	gvks := []schema.GroupVersionKind{
		types.GvkOf(&apiv2.ApisixRoute{}),
		types.GvkOf(&apiv2.ApisixGlobalRule{}),
		types.GvkOf(&apiv2.ApisixPluginConfig{}),
		types.GvkOf(&apiv2.ApisixTls{}),
		types.GvkOf(&apiv2.ApisixConsumer{}),
		types.GvkOf(&apiv2.ApisixUpstream{}),
	}
	if utils.HasAPIResource(mgr, &netv1.Ingress{}) {
		gvks = append(gvks, types.GvkOf(&netv1.Ingress{}))
	}

	c := mgr.GetClient()
	readier.RegisterGVK(readiness.GVKConfig{
		GVKs: gvks,
		Filter: readiness.GVKFilter(func(obj *unstructured.Unstructured) bool {
			icName, _, _ := unstructured.NestedString(obj.Object, "spec", "ingressClassName")
			ingressClass, _ := controller.FindMatchingIngressClassByName(context.Background(), c, log, icName, icgv.String())
			return ingressClass != nil
		}),
	})
	log.Info("Registered v2 GVKs for readiness checks", "gvks", gvks)
}

func registerGatewayAPIForReadinessGVK(mgr manager.Manager, readier readiness.ReadinessManager, log logr.Logger) {
	gvks := []schema.GroupVersionKind{}
	if utils.HasAPIResource(mgr, &gatewayv1.HTTPRoute{}) {
		gvks = append(gvks, types.GvkOf(&gatewayv1.HTTPRoute{}))
	}
	if utils.HasAPIResource(mgr, &gatewayv1.GRPCRoute{}) {
		gvks = append(gvks, types.GvkOf(&gatewayv1.GRPCRoute{}))
	}
	if utils.HasAPIResource(mgr, &gatewayv1alpha2.TCPRoute{}) {
		gvks = append(gvks, types.GvkOf(&gatewayv1alpha2.TCPRoute{}))
	}
	if utils.HasAPIResource(mgr, &gatewayv1alpha2.UDPRoute{}) {
		gvks = append(gvks, types.GvkOf(&gatewayv1alpha2.UDPRoute{}))
	}
	if utils.HasAPIResource(mgr, &gatewayv1alpha2.TLSRoute{}) {
		gvks = append(gvks, types.GvkOf(&gatewayv1alpha2.TLSRoute{}))
	}
	if len(gvks) == 0 {
		return
	}

	readier.RegisterGVK(readiness.GVKConfig{
		GVKs: gvks,
	})
	log.Info("Registered Gateway API GVKs for readiness checks", "gvks", gvks)
}

func registerV1alpha1ForReadinessGVK(mgr manager.Manager, readier readiness.ReadinessManager, log logr.Logger) {
	gvks := []schema.GroupVersionKind{}

	for _, resource := range []client.Object{
		&v1alpha1.Consumer{},
	} {
		if utils.HasAPIResource(mgr, resource) {
			gvks = append(gvks, types.GvkOf(resource))
		}
	}
	if len(gvks) == 0 {
		return
	}
	c := mgr.GetClient()
	readier.RegisterGVK(readiness.GVKConfig{
		GVKs: gvks,
=======
		},
	} {
		if utils.HasAPIResource(mgr, resource) {
			runnables = append(runnables, controller)
		} else {
			setupLog.Info("Skipping indexer setup, API not found in cluster", "api", utils.FormatGVK(resource))
		}
	}
	return runnables, nil
}

func registerReadiness(mgr manager.Manager, readier readiness.ReadinessManager) {
	log := ctrl.LoggerFrom(context.Background()).WithName("readiness")

	registerAPIv2ForReadiness(mgr, log, readier)
	if !config.ControllerConfig.DisableGatewayAPI {
		registerGatewayAPIForReadiness(mgr, log, readier)
	}
	registerAPIv1alpha1ForReadiness(mgr, log, readier)
}

func registerGatewayAPIForReadiness(
	mgr manager.Manager,
	log logr.Logger,
	readier readiness.ReadinessManager,
) {
	var installed []schema.GroupVersionKind
	for _, resource := range []client.Object{
		&gatewayv1.HTTPRoute{},
		&gatewayv1.GRPCRoute{},
		&gatewayv1alpha2.TCPRoute{},
		&gatewayv1alpha2.UDPRoute{},
		&gatewayv1alpha2.TLSRoute{},
	} {
		gvk := types.GvkOf(resource)
		if utils.HasAPIResource(mgr, resource) {
			installed = append(installed, gvk)
		} else {
			log.Info("Skipping readiness registration, API not found", "gvk", gvk)
		}
	}
	if len(installed) == 0 {
		return
	}

	readier.RegisterGVK(readiness.GVKConfig{GVKs: installed})
}

func registerAPIv2ForReadiness(
	mgr manager.Manager,
	log logr.Logger,
	readier readiness.ReadinessManager,
) {
	var installed []schema.GroupVersionKind
	for _, resource := range []client.Object{
		&netv1.Ingress{},
		&apiv2.ApisixRoute{},
		&apiv2.ApisixGlobalRule{},
		&apiv2.ApisixPluginConfig{},
		&apiv2.ApisixTls{},
		&apiv2.ApisixConsumer{},
		&apiv2.ApisixUpstream{},
	} {
		gvk := types.GvkOf(resource)
		if utils.HasAPIResource(mgr, resource) {
			installed = append(installed, gvk)
		} else {
			log.Info("Skipping readiness registration, API not found", "gvk", gvk)
		}
	}

	if len(installed) == 0 {
		return
	}

	readier.RegisterGVK(readiness.GVKConfig{
		GVKs: installed,
		Filter: readiness.GVKFilter(func(obj *unstructured.Unstructured) bool {
			icName, _, _ := unstructured.NestedString(obj.Object, "spec", "ingressClassName")
			ingressClass, _ := controller.FindMatchingIngressClassByName(context.Background(), mgr.GetClient(), log, icName)
			return ingressClass != nil
		}),
	})
}

func registerAPIv1alpha1ForReadiness(
	mgr manager.Manager,
	log logr.Logger,
	readier readiness.ReadinessManager,
) {
	var installed []schema.GroupVersionKind
	for _, resource := range []client.Object{
		&v1alpha1.Consumer{},
	} {
		gvk := types.GvkOf(resource)
		if utils.HasAPIResource(mgr, resource) {
			installed = append(installed, gvk)
		} else {
			log.Info("Skipping readiness registration, API not found", "gvk", gvk)
		}
	}
	if len(installed) == 0 {
		return
	}

	readier.RegisterGVK(readiness.GVKConfig{
		GVKs: installed,
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
		Filter: readiness.GVKFilter(func(obj *unstructured.Unstructured) bool {
			consumer := &v1alpha1.Consumer{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, consumer); err != nil {
				return false
			}
<<<<<<< HEAD
			return controller.MatchConsumerGatewayRef(context.Background(), c, log, consumer)
		}),
	})
	log.Info("Registered v1alpha1 GVKs for readiness checks", "gvks", gvks)
=======
			return controller.MatchConsumerGatewayRef(context.Background(), mgr.GetClient(), log, consumer)
		}),
	})
>>>>>>> 4f9cd000 (feat: support disable gateway-api (#2672))
}
