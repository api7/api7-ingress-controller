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
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

const (
	ProviderTypeAPI7EE = "api7ee"

	RetryBaseDelay = 1 * time.Second
	RetryMaxDelay  = 1000 * time.Second
)

// api7eeProvider owns AIC's own view of what should be live: which Kubernetes resource
// targets which GatewayProxy config (configManager) and the merged, translated resource
// snapshot per config (store). It builds the input the adc client package needs and hands
// it over on every call; the client package holds none of this state itself.
type api7eeProvider struct {
	sync.Mutex

	translator *translator.Translator

	store         *cache.Store
	configManager *common.ConfigManager[types.NamespacedNameKind, adctypes.Config]
	debugProvider *common.ADCDebugProvider

	// syncLocks serializes, per cacheKey, reading that GatewayProxy's current resource
	// snapshot together with pushing it
	syncLocks *common.KeyedMutex

	updater         status.Updater
	statusUpdateMap map[types.NamespacedNameKind][]string

	readier readiness.ReadinessManager

	provider.Options

	syncCh chan struct{}

	startUpSync atomic.Bool

	client *adcclient.Client
	log    logr.Logger
}

func New(log logr.Logger, updater status.Updater, readier readiness.ReadinessManager, opts ...provider.Option) (provider.Provider, error) {
	o := provider.Options{}
	o.ApplyOptions(opts)
	if o.DefaultBackendMode == "" {
		o.DefaultBackendMode = ProviderTypeAPI7EE
	}

	cli, err := adcclient.New(log, o.DefaultBackendMode, o.SyncTimeout)
	if err != nil {
		return nil, err
	}

	logger := log.WithName("provider")

	store := cache.NewStore(logger)
	configManager := common.NewConfigManager[types.NamespacedNameKind, adctypes.Config]()

	return &api7eeProvider{
		client:        cli,
		store:         store,
		configManager: configManager,
		debugProvider: common.NewADCDebugProvider(store, configManager),
		syncLocks:     common.NewKeyedMutex(),
		Options:       o,
		translator:    translator.NewTranslator(log, o.ListenerPortMatchMode),
		updater:       updater,
		readier:       readier,
		syncCh:        make(chan struct{}, 1),
		log:           logger,
	}, nil
}

func (d *api7eeProvider) Register(pathPrefix string, mux *http.ServeMux) {
	d.debugProvider.SetupHandler(pathPrefix, mux)
}

func (d *api7eeProvider) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	d.log.V(1).Info("updating object", "object", obj)
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
	case *gatewayv1.TCPRoute:
		result, err = d.translator.TranslateTCPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.UDPRoute:
		result, err = d.translator.TranslateUDPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.TLSRoute:
		result, err = d.translator.TranslateTLSRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.GRPCRoute:
		result, err = d.translator.TranslateGRPCRoute(tctx, t.DeepCopy())
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

	resources := &adctypes.Resources{
		GlobalRules:    result.GlobalRules,
		PluginMetadata: result.PluginMetadata,
		Services:       result.Services,
		SSLs:           result.SSL,
		Consumers:      result.Consumers,
	}
	labels := label.GenLabel(obj)

	evicted, err := d.applyResourceState(rk, configs, resourceTypes, resources, labels)
	if err != nil {
		return err
	}

	if !d.startUpSync.Load() {
		d.log.V(1).Info("startup synchronization not completed, skip sync", "object", obj)
		return nil
	}

	// A GatewayProxy this resource no longer targets must still lose this resource's
	// contribution on the data plane. That push is best-effort, the same as the deferred
	// path's: only the periodic sync ever surfaces its failures as a status update.
	d.syncEvictedConfigsNow(ctx, evicted, resourceTypes, labels)

	return d.pushConfigsNow(ctx, configs, resourceTypes, resources, labels)
}

func (d *api7eeProvider) Delete(ctx context.Context, obj client.Object) error {
	d.log.V(1).Info("deleting object", "object", obj)

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute, *apiv2.ApisixRoute, *gatewayv1.GRPCRoute, *gatewayv1.TCPRoute, *gatewayv1.UDPRoute, *gatewayv1.TLSRoute:
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

	removed, err := d.removeResourceState(nnk, resourceTypes, labels)
	if err != nil {
		return err
	}

	// A deleted resource always pushes right away rather than waiting for the next
	// scheduled round, and -- like the deferred path -- only logs a push failure instead
	// of surfacing it, since there is no per-object status to report it against once the
	// object itself is gone.
	d.syncEvictedConfigsNow(ctx, removed, resourceTypes, labels)
	return nil
}

// applyResourceState upserts a resource's config associations and its contribution to each
// target config's cached resource snapshot -- the AIC-side bookkeeping the adc client
// package no longer holds itself. It returns the configs this resource no longer
// references (if any), so the caller can push their eviction right away instead of
// waiting for the next scheduled sync round.
func (d *api7eeProvider) applyResourceState(
	rk types.NamespacedNameKind,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	resources *adctypes.Resources,
	labels map[string]string,
) (map[types.NamespacedNameKind]adctypes.Config, error) {
	d.Lock()
	defer d.Unlock()

	evicted := d.configManager.Update(rk, configs)
	if err := d.evictFromStore(evicted, resourceTypes, labels); err != nil {
		return nil, err
	}
	for _, cfg := range configs {
		if err := d.store.Insert(cfg.Name, resourceTypes, resources, labels); err != nil {
			return nil, fmt.Errorf("store insert failed for config %s: %w", cfg.Name, err)
		}
	}
	return evicted, nil
}

// removeResourceState forgets a resource's config associations and evicts its contribution
// from each config it used to reference, returning those configs so an immediate-push
// caller (see syncEvictedConfigsNow) knows what to push right away.
func (d *api7eeProvider) removeResourceState(
	rk types.NamespacedNameKind,
	resourceTypes []string,
	labels map[string]string,
) (map[types.NamespacedNameKind]adctypes.Config, error) {
	d.Lock()
	defer d.Unlock()

	evicted := d.configManager.Get(rk)
	d.configManager.Delete(rk)
	if err := d.evictFromStore(evicted, resourceTypes, labels); err != nil {
		return nil, err
	}
	return evicted, nil
}

// evictFromStore deletes a resource's contribution from each of the given configs' cached
// snapshots. Callers must already hold d.Lock.
func (d *api7eeProvider) evictFromStore(
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	labels map[string]string,
) error {
	for _, cfg := range configs {
		if err := d.store.Delete(cfg.Name, resourceTypes, labels); err != nil {
			return fmt.Errorf("store delete failed for config %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// syncConfigNow reads name's current data (via build, called only once this cacheKey's
// lock is actually held) and pushes it -- one atomic read-then-push step per cacheKey, so
// whichever caller is granted the lock decides what to push only once it holds it: nothing
// it sends can already be stale relative to whatever the other caller committed to the
// store before losing the race for the same key. See common.KeyedMutex.
func (d *api7eeProvider) syncConfigNow(
	ctx context.Context,
	name string,
	build func() (adcclient.SyncInput, error),
) (types.ADCExecutionErrors, error) {
	unlock := d.syncLocks.Lock(name)
	defer unlock()

	input, err := build()
	if err != nil {
		return types.ADCExecutionErrors{}, err
	}
	failedMap, err := d.client.Sync(ctx, []adcclient.SyncInput{input})
	return failedMap[name], err
}

// pushConfigsNow pushes resources immediately to every one of the given configs, through
// the same per-cacheKey lock the periodic sync uses, and reports the combined push error.
// This is the immediate half of Update: it only runs once startup synchronization has
// completed, mirroring the old Client.Update. Before that, applyResourceState alone is
// enough -- the startup sync and the periodic ticker eventually push it.
func (d *api7eeProvider) pushConfigsNow(
	ctx context.Context,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	resources *adctypes.Resources,
	labels map[string]string,
) error {
	var errs []error
	for _, cfg := range configs {
		_, err := d.syncConfigNow(ctx, cfg.Name, func() (adcclient.SyncInput, error) {
			mergedResources, err := common.WithMergedGlobalRules(d.store, cfg.Name, resourceTypes, resources)
			if err != nil {
				return adcclient.SyncInput{}, err
			}
			return adcclient.SyncInput{
				Name:          cfg.Name,
				Config:        cfg,
				Resources:     mergedResources,
				ResourceTypes: resourceTypes,
				Labels:        common.ResourceKeyLabels(labels),
			}, nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("config %s: %w", cfg.Name, err))
		}
	}
	return errors.Join(errs...)
}

// syncEvictedConfigsNow pushes an empty resource set for each of the given configs
// immediately, instead of waiting for the next scheduled sync round -- through the same
// per-cacheKey lock the periodic sync uses, so it can never race a periodic round for the
// same GatewayProxy. Failures are logged, not surfaced as a status update -- this mirrors
// the deferred path, which only reports through the next scheduled sync round.
func (d *api7eeProvider) syncEvictedConfigsNow(
	ctx context.Context,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	labels map[string]string,
) {
	for _, cfg := range configs {
		_, err := d.syncConfigNow(ctx, cfg.Name, func() (adcclient.SyncInput, error) {
			resources, err := common.WithMergedGlobalRules(d.store, cfg.Name, resourceTypes, &adctypes.Resources{})
			if err != nil {
				return adcclient.SyncInput{}, err
			}
			return adcclient.SyncInput{
				Name:          cfg.Name,
				Config:        cfg,
				Resources:     resources,
				ResourceTypes: resourceTypes,
				Labels:        common.ResourceKeyLabels(labels),
			}, nil
		})
		if err != nil {
			d.log.Error(err, "failed to sync deleted config", "config", cfg)
		}
	}
}

func (d *api7eeProvider) Start(ctx context.Context) error {
	d.readier.WaitReady(ctx, 5*time.Minute)

	d.startUpSync.Store(true)
	d.log.Info("Performing startup synchronization")
	if err := d.sync(ctx); err != nil {
		d.log.Error(err, "failed to sync for startup")
	}

	initalSyncDelay := d.InitSyncDelay
	if initalSyncDelay > 0 {
		time.AfterFunc(initalSyncDelay, func() {
			if err := d.sync(ctx); err != nil {
				d.log.Error(err, "failed to sync for startup")
				return
			}
		})
	}

	if d.SyncPeriod < 1 {
		return nil
	}
	ticker := time.NewTicker(d.SyncPeriod)
	defer ticker.Stop()

	retrier := common.NewRetrier(common.NewExponentialBackoff(RetryBaseDelay, RetryMaxDelay))

	for {
		select {
		case <-d.syncCh:
		case <-retrier.C():
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}

		if err := d.sync(ctx); err != nil {
			d.log.Error(err, "failed to sync")
			retrier.Next()
		} else {
			retrier.Reset()
		}
	}
}

func (d *api7eeProvider) syncNotify() {
	select {
	case d.syncCh <- struct{}{}:
	default:
	}
}

// sync pushes every GatewayProxy AIC currently knows about, config by config -- each
// one's current resource snapshot is only read once syncConfigNow actually holds that
// cacheKey's lock, so a slow round can never push a snapshot that was already stale by the
// time its turn came up. All of this round's results are still collected into one
// statusesMap and handed to handleADCExecutionErrors together, exactly as a single batched
// sync would: that logic diffs against last round's full picture, not per-config.
func (d *api7eeProvider) sync(ctx context.Context) error {
	configs := d.configManager.List()

	statusesMap := map[string]types.ADCExecutionErrors{}
	var errs []error
	for _, config := range configs {
		execErrs, err := d.syncConfigNow(ctx, config.Name, func() (adcclient.SyncInput, error) {
			resources, err := d.store.GetResources(config.Name)
			if err != nil {
				return adcclient.SyncInput{}, fmt.Errorf("failed to get resources from store: %w", err)
			}
			return adcclient.SyncInput{Name: config.Name, Config: config, Resources: resources}, nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("config %s: %w", config.Name, err))
		}
		if len(execErrs.Errors) > 0 {
			statusesMap[config.Name] = execErrs
		}
	}

	d.handleADCExecutionErrors(statusesMap)
	return errors.Join(errs...)
}

func (d *api7eeProvider) handleADCExecutionErrors(statusesMap map[string]types.ADCExecutionErrors) {
	statusUpdateMap := d.resolveADCExecutionErrors(statusesMap)
	d.handleStatusUpdate(statusUpdateMap)
	d.log.V(1).Info("handled ADC execution errors", "status_record", statusesMap, "status_update", statusUpdateMap)
}

func (d *api7eeProvider) NeedLeaderElection() bool {
	return true
}

// updateConfigForGatewayProxy update config for all referrers of the GatewayProxy
func (d *api7eeProvider) updateConfigForGatewayProxy(tctx *provider.TranslateContext, gp *v1alpha1.GatewayProxy) error {
	config, err := d.translator.TranslateGatewayProxyToConfig(tctx, gp, d.DefaultResolveEndpoints)
	if err != nil {
		return err
	}

	nnk := utils.NamespacedNameKind(gp)
	if config == nil {
		d.Lock()
		d.configManager.DeleteConfig(nnk)
		d.Unlock()
		return nil
	}

	referrers := tctx.GatewayProxyReferrers[utils.NamespacedName(gp)]
	d.Lock()
	d.configManager.SetConfigRefs(nnk, referrers)
	d.configManager.UpdateConfig(nnk, *config)
	d.Unlock()
	d.syncNotify()
	return nil
}

func (d *api7eeProvider) buildConfig(tctx *provider.TranslateContext, nnk types.NamespacedNameKind) (map[types.NamespacedNameKind]adctypes.Config, error) {
	configs := make(map[types.NamespacedNameKind]adctypes.Config, len(tctx.ResourceParentRefs[nnk]))
	for _, gp := range tctx.GatewayProxies {
		config, err := d.translator.TranslateGatewayProxyToConfig(tctx, &gp, d.DefaultResolveEndpoints)
		if err != nil {
			return nil, err
		}
		configs[utils.NamespacedNameKind(&gp)] = *config
	}
	return configs, nil
}
