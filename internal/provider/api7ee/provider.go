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

package api7ee

import (
	"context"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

const ProviderTypeAPI7EE = "api7ee"

type api7eeProvider struct {
	translator *translator.Translator

	updater         status.Updater
	statusUpdateMap map[types.NamespacedNameKind][]string

	readier readiness.ReadinessManager

	provider.Options

	syncCh chan struct{}

	client *adcclient.Client
}

func New(updater status.Updater, readier readiness.ReadinessManager, opts ...provider.Option) (provider.Provider, error) {
	o := provider.Options{}
	o.ApplyOptions(opts)
	if o.BackendMode == "" {
		o.BackendMode = ProviderTypeAPI7EE
	}

	cli, err := adcclient.New(o.BackendMode)
	if err != nil {
		return nil, err
	}

	return &api7eeProvider{
		client:     cli,
		Options:    o,
		translator: &translator.Translator{},
		updater:    updater,
		readier:    readier,
		syncCh:     make(chan struct{}, 1),
	}, nil
}

func (d *api7eeProvider) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	log.Debugw("updating object", zap.Any("object", obj))
	var (
		result        *translator.TranslateResult
		resourceTypes []string
		err           error
	)

	rk := utils.NamespacedNameKind(obj)

	switch t := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "ssl", "plugin_metadata")
	case *networkingv1.Ingress:
		result, err = d.translator.TranslateIngress(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service", "ssl")
	case *v1alpha1.Consumer:
		result, err = d.translator.TranslateConsumerV1alpha1(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
	case *networkingv1.IngressClass:
		result, err = d.translator.TranslateIngressClass(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "plugin_metadata")
	case *networkingv1beta1.IngressClass:
		cp := pkgutils.ConvertToIngressClassV1(t.DeepCopy())
		result, err = d.translator.TranslateIngressClass(tctx, cp)
		resourceTypes = append(resourceTypes, "global_rule", "plugin_metadata")
	case *apiv2.ApisixRoute:
		result, err = d.translator.TranslateApisixRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *apiv2.ApisixGlobalRule:
		result, err = d.translator.TranslateApisixGlobalRule(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule")
	case *apiv2.ApisixTls:
		result, err = d.translator.TranslateApisixTls(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "ssl")
	case *apiv2.ApisixConsumer:
		result, err = d.translator.TranslateApisixConsumer(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
	case *v1alpha1.GatewayProxy:
		return d.updateConfigForGatewayProxy(tctx, t)
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	configs, err := d.buildConfig(tctx, rk)
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		return nil
	}

	nnk := utils.NamespacedNameKind(obj)

	task := adcclient.Task{
		Key:           nnk,
		Name:          nnk.String(),
		Labels:        label.GenLabel(obj),
		Configs:       configs,
		ResourceTypes: resourceTypes,
		Resources: &adctypes.Resources{
			GlobalRules:    result.GlobalRules,
			PluginMetadata: result.PluginMetadata,
			Services:       result.Services,
			SSLs:           result.SSL,
			Consumers:      result.Consumers,
		},
	}

	return d.client.Update(ctx, task)
}

func (d *api7eeProvider) Delete(ctx context.Context, obj client.Object) error {
	log.Debugw("deleting object", zap.Any("object", obj))

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute, *apiv2.ApisixRoute:
		resourceTypes = append(resourceTypes, "service")
		labels = label.GenLabel(obj)
	case *gatewayv1.Gateway:
		// delete all resources
	case *networkingv1.Ingress:
		resourceTypes = append(resourceTypes, "service", "ssl")
		labels = label.GenLabel(obj)
	case *v1alpha1.Consumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	case *networkingv1.IngressClass, *networkingv1beta1.IngressClass:
		// delete all resources
	case *apiv2.ApisixGlobalRule:
		resourceTypes = append(resourceTypes, "global_rule")
		labels = label.GenLabel(obj)
	case *apiv2.ApisixTls:
		resourceTypes = append(resourceTypes, "ssl")
		labels = label.GenLabel(obj)
	case *apiv2.ApisixConsumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	}

	nnk := utils.NamespacedNameKind(obj)
	return d.client.Delete(ctx, adcclient.Task{
		Key:           nnk,
		Name:          nnk.String(),
		Labels:        labels,
		ResourceTypes: resourceTypes,
	})
}

func (d *api7eeProvider) Start(ctx context.Context) error {
	d.readier.WaitReady(ctx, 5*time.Minute)

	initalSyncDelay := d.InitSyncDelay
	if initalSyncDelay > 0 {
		time.AfterFunc(initalSyncDelay, func() {
			if err := d.sync(ctx); err != nil {
				log.Error(err)
				return
			}
		})
	}

	if d.SyncPeriod < 1 {
		return nil
	}
	ticker := time.NewTicker(d.SyncPeriod)
	defer ticker.Stop()
	for {
		synced := false
		select {
		case <-d.syncCh:
			synced = true
		case <-ticker.C:
			synced = true
		case <-ctx.Done():
			return nil
		}
		if synced {
			if err := d.sync(ctx); err != nil {
				log.Error(err)
			}
		}
	}
}

func (d *api7eeProvider) syncNotify() {
	select {
	case d.syncCh <- struct{}{}:
	default:
	}
}

func (d *api7eeProvider) sync(ctx context.Context) error {
	statusesMap, err := d.client.Sync(ctx)
	d.handleADCExecutionErrors(statusesMap)
	return err
}

func (d *api7eeProvider) handleADCExecutionErrors(statusesMap map[string]types.ADCExecutionErrors) {
	statusUpdateMap := d.resolveADCExecutionErrors(statusesMap)
	d.handleStatusUpdate(statusUpdateMap)
	log.Debugw("handled ADC execution errors", zap.Any("status_record", statusesMap), zap.Any("status_update", statusUpdateMap))
}

func (d *api7eeProvider) NeedLeaderElection() bool {
	return true
}

// updateConfigForGatewayProxy update config for all referrers of the GatewayProxy
func (d *api7eeProvider) updateConfigForGatewayProxy(tctx *provider.TranslateContext, gp *v1alpha1.GatewayProxy) error {
	config, err := d.translator.TranslateGatewayProxyToConfig(tctx, gp, d.ResolveEndpoints)
	if err != nil {
		return err
	}

	nnk := utils.NamespacedNameKind(gp)
	if config == nil {
		d.client.ConfigManager.DeleteConfig(nnk)
		return nil
	}
	referrers := tctx.GatewayProxyReferrers[utils.NamespacedName(gp)]
	d.client.ConfigManager.SetConfigRefs(nnk, referrers)
	d.client.ConfigManager.UpdateConfig(nnk, *config)
	d.syncNotify()
	return nil
}

func (d *api7eeProvider) buildConfig(tctx *provider.TranslateContext, nnk types.NamespacedNameKind) (map[types.NamespacedNameKind]adctypes.Config, error) {
	configs := make(map[types.NamespacedNameKind]adctypes.Config, len(tctx.ResourceParentRefs[nnk]))
	for _, gp := range tctx.GatewayProxies {
		config, err := d.translator.TranslateGatewayProxyToConfig(tctx, &gp, d.ResolveEndpoints)
		if err != nil {
			return nil, err
		}
		configs[utils.NamespacedNameKind(&gp)] = *config
	}
	return configs, nil
}
