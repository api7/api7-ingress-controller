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

	netv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	types "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/pkg/utils"
)

// K8s
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

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
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants,verbs=list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants/status,verbs=get;update

// Networking
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch

type Controller interface {
	SetupWithManager(mgr manager.Manager) error
}

func setupControllers(ctx context.Context, mgr manager.Manager, pro provider.Provider, updater status.Updater, readier readiness.ReadinessManager) ([]Controller, error) {
	if err := indexer.SetupIndexer(mgr); err != nil {
		return nil, err
	}

	setupLog := ctrl.LoggerFrom(ctx).WithName("setup")
	var controllers []Controller

	icgvk := types.GvkOf(&v1.IngressClass{})
	if !utils.HasAPIResource(mgr, &v1.IngressClass{}) {
		setupLog.Info("IngressClass v1 not found, falling back to IngressClass v1beta1")
		icgvk = types.GvkOf(&v1beta1.IngressClass{})
		controllers = append(controllers, &controller.IngressClassV1beta1Reconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("IngressClass"),
			Provider: pro,
		})
	}

	// Gateway API Controllers - conditional registration based on API availability
	for resource, controller := range map[client.Object]Controller{
		&gatewayv1.GatewayClass{}: &controller.GatewayClassReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("GatewayClass"),
			Updater: updater,
		},
		&gatewayv1.Gateway{}: &controller.GatewayReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Gateway"),
			Provider: pro,
			Updater:  updater,
		},
		&gatewayv1.HTTPRoute{}: &controller.HTTPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("HTTPRoute"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&v1alpha1.Consumer{}: &controller.ConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Consumer"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&v1.Ingress{}: &controller.IngressReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Ingress"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
		},
		&v1.IngressClass{}: &controller.IngressClassReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("IngressClass"),
			Provider: pro,
		},
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
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("GatewayProxy"),
			Provider: pro,
			ICGVK:    icgvk,
		},
		// APISIX v2 Controllers - always register these as they are core to the controller
		&controller.ApisixGlobalRuleReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixGlobalRule"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGVK:    icgvk,
		},
		&controller.ApisixRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixRoute"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGVK:    icgvk,
		},
		&controller.ApisixConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixConsumer"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGVK:    icgvk,
		},
		&controller.ApisixPluginConfigReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixPluginConfig"),
			Updater: updater,
			ICGVK:   icgvk,
		},
		&controller.ApisixTlsReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixTls"),
			Provider: pro,
			Updater:  updater,
			Readier:  readier,
			ICGVK:    icgvk,
		},
		&controller.ApisixUpstreamReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixUpstream"),
			Updater: updater,
			ICGVK:   icgvk,
		},
	}...)

	setupLog.Info("Controllers setup completed", "total_controllers", len(controllers))
	return controllers, nil
}

func registerReadinessGVK(mgr manager.Manager, readier readiness.ReadinessManager) {
	c := mgr.GetClient()
	log := ctrl.LoggerFrom(context.Background()).WithName("readiness")

	icgvk := types.GvkOf(&v1.IngressClass{})
	if !utils.HasAPIResource(mgr, &v1.IngressClass{}) {
		icgvk = types.GvkOf(&v1beta1.IngressClass{})
	}

	readier.RegisterGVK([]readiness.GVKConfig{
		{
			GVKs: []schema.GroupVersionKind{
				types.GvkOf(&gatewayv1.HTTPRoute{}),
			},
		},
		{
			GVKs: []schema.GroupVersionKind{
				types.GvkOf(&netv1.Ingress{}),
				types.GvkOf(&apiv2.ApisixRoute{}),
				types.GvkOf(&apiv2.ApisixGlobalRule{}),
				types.GvkOf(&apiv2.ApisixPluginConfig{}),
				types.GvkOf(&apiv2.ApisixTls{}),
				types.GvkOf(&apiv2.ApisixConsumer{}),
			},
			Filter: readiness.GVKFilter(func(obj *unstructured.Unstructured) bool {
				icName, _, _ := unstructured.NestedString(obj.Object, "spec", "ingressClassName")
				ingressClass, _ := controller.GetIngressClass(context.Background(), c, log, icName, icgvk.Version)
				return ingressClass != nil
			}),
		},
		{
			GVKs: []schema.GroupVersionKind{
				types.GvkOf(&v1alpha1.Consumer{}),
			},
			Filter: readiness.GVKFilter(func(obj *unstructured.Unstructured) bool {
				consumer := &v1alpha1.Consumer{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, consumer); err != nil {
					return false
				}
				return controller.MatchConsumerGatewayRef(context.Background(), c, log, consumer)
			}),
		},
	}...)
}
