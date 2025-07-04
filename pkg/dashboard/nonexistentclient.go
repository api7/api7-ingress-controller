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

package dashboard

import (
	"context"

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard/cache"
)

type nonExistentCluster struct {
	embedDummyResourceImplementer
}

func newNonExistentCluster() *nonExistentCluster {
	return &nonExistentCluster{
		embedDummyResourceImplementer{
			route:          &dummyRoute{},
			ssl:            &dummySSL{},
			service:        &dummyService{},
			streamRoute:    &dummyStreamRoute{},
			globalRule:     &dummyGlobalRule{},
			consumer:       &dummyConsumer{},
			plugin:         &dummyPlugin{},
			schema:         &dummySchema{},
			pluginConfig:   &dummyPluginConfig{},
			pluginMetadata: &dummyPluginMetadata{},
		},
	}
}

type embedDummyResourceImplementer struct {
	route          Route
	ssl            SSL
	service        Service
	streamRoute    StreamRoute
	globalRule     GlobalRule
	consumer       Consumer
	plugin         Plugin
	schema         Schema
	pluginConfig   PluginConfig
	pluginMetadata PluginMetadata
	validator      APISIXSchemaValidator
}

type dummyRoute struct{}

func (f *dummyRoute) Get(_ context.Context, _ string) (*v1.Route, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyRoute) List(_ context.Context, _ ...any) ([]*v1.Route, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyRoute) Create(_ context.Context, _ *v1.Route) (*v1.Route, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyRoute) Delete(_ context.Context, _ *v1.Route) error {
	return ErrClusterNotExist
}

func (f *dummyRoute) Update(_ context.Context, _ *v1.Route) (*v1.Route, error) {
	return nil, ErrClusterNotExist
}

type dummySSL struct{}

func (f *dummySSL) Get(_ context.Context, _ string) (*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySSL) List(_ context.Context, _ ...any) ([]*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySSL) Create(_ context.Context, _ *v1.Ssl) (*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySSL) Delete(_ context.Context, _ *v1.Ssl) error {
	return ErrClusterNotExist
}

func (f *dummySSL) Update(_ context.Context, _ *v1.Ssl) (*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

type dummyService struct{}

func (f *dummyService) Get(_ context.Context, _ string) (*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyService) List(_ context.Context, _ ...any) ([]*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyService) Create(_ context.Context, _ *v1.Service) (*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyService) Delete(_ context.Context, _ *v1.Service) error {
	return ErrClusterNotExist
}

func (f *dummyService) Update(_ context.Context, _ *v1.Service) (*v1.Service, error) {
	return nil, ErrClusterNotExist
}

type dummyStreamRoute struct{}

func (f *dummyStreamRoute) Get(_ context.Context, _ string) (*v1.StreamRoute, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyStreamRoute) List(_ context.Context) ([]*v1.StreamRoute, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyStreamRoute) Create(_ context.Context, _ *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyStreamRoute) Delete(_ context.Context, _ *v1.StreamRoute) error {
	return ErrClusterNotExist
}

func (f *dummyStreamRoute) Update(_ context.Context, _ *v1.StreamRoute) (*v1.StreamRoute, error) {
	return nil, ErrClusterNotExist
}

type dummyGlobalRule struct{}

func (f *dummyGlobalRule) Get(_ context.Context, _ string) (*v1.GlobalRule, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyGlobalRule) List(_ context.Context) ([]*v1.GlobalRule, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyGlobalRule) Create(_ context.Context, _ *v1.GlobalRule) (*v1.GlobalRule, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyGlobalRule) Delete(_ context.Context, _ *v1.GlobalRule) error {
	return ErrClusterNotExist
}

func (f *dummyGlobalRule) Update(_ context.Context, _ *v1.GlobalRule) (*v1.GlobalRule, error) {
	return nil, ErrClusterNotExist
}

type dummyConsumer struct{}

func (f *dummyConsumer) Get(_ context.Context, _ string) (*v1.Consumer, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyConsumer) List(_ context.Context) ([]*v1.Consumer, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyConsumer) Create(_ context.Context, _ *v1.Consumer) (*v1.Consumer, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyConsumer) Delete(_ context.Context, _ *v1.Consumer) error {
	return ErrClusterNotExist
}

func (f *dummyConsumer) Update(_ context.Context, _ *v1.Consumer) (*v1.Consumer, error) {
	return nil, ErrClusterNotExist
}

type dummyPlugin struct{}

func (f *dummyPlugin) List(_ context.Context) ([]string, error) {
	return nil, ErrClusterNotExist
}

type dummySchema struct{}

func (f *dummySchema) GetPluginSchema(_ context.Context, _ string) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySchema) GetRouteSchema(_ context.Context) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySchema) GetUpstreamSchema(_ context.Context) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySchema) GetConsumerSchema(_ context.Context) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySchema) GetSslSchema(_ context.Context) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySchema) GetPluginConfigSchema(_ context.Context) (*v1.Schema, error) {
	return nil, ErrClusterNotExist
}

type dummyPluginConfig struct{}

func (f *dummyPluginConfig) Get(_ context.Context, _ string) (*v1.PluginConfig, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyPluginConfig) List(_ context.Context) ([]*v1.PluginConfig, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyPluginConfig) Create(_ context.Context, _ *v1.PluginConfig) (*v1.PluginConfig, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyPluginConfig) Delete(_ context.Context, _ *v1.PluginConfig) error {
	return ErrClusterNotExist
}

func (f *dummyPluginConfig) Update(_ context.Context, _ *v1.PluginConfig) (*v1.PluginConfig, error) {
	return nil, ErrClusterNotExist
}

type dummyPluginMetadata struct {
}

func (f *dummyPluginMetadata) Get(_ context.Context, _ string) (*v1.PluginMetadata, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyPluginMetadata) List(_ context.Context) ([]*v1.PluginMetadata, error) {
	return nil, ErrClusterNotExist
}
func (f *dummyPluginMetadata) Delete(_ context.Context, _ *v1.PluginMetadata) error {
	return ErrClusterNotExist
}
func (f *dummyPluginMetadata) Update(_ context.Context, _ *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	return nil, ErrClusterNotExist
}
func (f *dummyPluginMetadata) Create(_ context.Context, _ *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	return nil, ErrClusterNotExist
}

type dummyValidator struct{}

func newDummyValidator() APISIXSchemaValidator {
	return &dummyValidator{}
}

func (d *dummyValidator) ValidateHTTPPluginSchema(plugins v1.Plugins) (bool, error) {
	return true, nil
}

func (d *dummyValidator) ValidateStreamPluginSchema(plugins v1.Plugins) (bool, error) {
	return true, nil
}

func (nc *nonExistentCluster) Route() Route {
	return nc.route
}

func (nc *nonExistentCluster) SSL() SSL {
	return nc.ssl
}

func (nc *nonExistentCluster) Service() Service {
	return nc.service
}

func (nc *nonExistentCluster) StreamRoute() StreamRoute {
	return nc.streamRoute
}

func (nc *nonExistentCluster) GlobalRule() GlobalRule {
	return nc.globalRule
}

func (nc *nonExistentCluster) Consumer() Consumer {
	return nc.consumer
}

func (nc *nonExistentCluster) Plugin() Plugin {
	return nc.plugin
}

func (nc *nonExistentCluster) Validator() APISIXSchemaValidator {
	return nc.validator
}

func (nc *nonExistentCluster) PluginConfig() PluginConfig {
	return nc.pluginConfig
}

func (nc *nonExistentCluster) Schema() Schema {
	return nc.schema
}
func (nc *nonExistentCluster) PluginMetadata() PluginMetadata {
	return nc.pluginMetadata
}

func (nc *nonExistentCluster) HasSynced(_ context.Context) error {
	return nil
}

func (nc *nonExistentCluster) HealthCheck(_ context.Context) error {
	return nil
}

func (nc *nonExistentCluster) String() string {
	return "non-existent cluster"
}

type dummyCache struct{}

var _ cache.Cache = &dummyCache{}

func (c *dummyCache) InsertRoute(_ *v1.Route) error                    { return nil }
func (c *dummyCache) InsertSSL(_ *v1.Ssl) error                        { return nil }
func (c *dummyCache) InsertService(_ *v1.Service) error                { return nil }
func (c *dummyCache) InsertStreamRoute(_ *v1.StreamRoute) error        { return nil }
func (c *dummyCache) InsertGlobalRule(_ *v1.GlobalRule) error          { return nil }
func (c *dummyCache) InsertConsumer(_ *v1.Consumer) error              { return nil }
func (c *dummyCache) InsertSchema(_ *v1.Schema) error                  { return nil }
func (c *dummyCache) InsertPluginConfig(_ *v1.PluginConfig) error      { return nil }
func (c *dummyCache) GetRoute(_ string) (*v1.Route, error)             { return nil, cache.ErrNotFound }
func (c *dummyCache) GetSSL(_ string) (*v1.Ssl, error)                 { return nil, cache.ErrNotFound }
func (c *dummyCache) GetService(_ string) (*v1.Service, error)         { return nil, cache.ErrNotFound }
func (c *dummyCache) GetStreamRoute(_ string) (*v1.StreamRoute, error) { return nil, cache.ErrNotFound }
func (c *dummyCache) GetGlobalRule(_ string) (*v1.GlobalRule, error)   { return nil, cache.ErrNotFound }
func (c *dummyCache) GetConsumer(_ string) (*v1.Consumer, error)       { return nil, cache.ErrNotFound }
func (c *dummyCache) GetSchema(_ string) (*v1.Schema, error)           { return nil, cache.ErrNotFound }
func (c *dummyCache) GetPluginConfig(_ string) (*v1.PluginConfig, error) {
	return nil, cache.ErrNotFound
}

func (c *dummyCache) ListRoutes(...any) ([]*v1.Route, error)         { return nil, nil }
func (c *dummyCache) ListSSL(_ ...any) ([]*v1.Ssl, error)            { return nil, nil }
func (c *dummyCache) ListServices(...any) ([]*v1.Service, error)     { return nil, nil }
func (c *dummyCache) ListStreamRoutes() ([]*v1.StreamRoute, error)   { return nil, nil }
func (c *dummyCache) ListGlobalRules() ([]*v1.GlobalRule, error)     { return nil, nil }
func (c *dummyCache) ListConsumers() ([]*v1.Consumer, error)         { return nil, nil }
func (c *dummyCache) ListSchema() ([]*v1.Schema, error)              { return nil, nil }
func (c *dummyCache) ListPluginConfigs() ([]*v1.PluginConfig, error) { return nil, nil }

func (c *dummyCache) DeleteRoute(_ *v1.Route) error                       { return nil }
func (c *dummyCache) DeleteSSL(_ *v1.Ssl) error                           { return nil }
func (c *dummyCache) DeleteService(_ *v1.Service) error                   { return nil }
func (c *dummyCache) DeleteStreamRoute(_ *v1.StreamRoute) error           { return nil }
func (c *dummyCache) DeleteGlobalRule(_ *v1.GlobalRule) error             { return nil }
func (c *dummyCache) DeleteConsumer(_ *v1.Consumer) error                 { return nil }
func (c *dummyCache) DeleteSchema(_ *v1.Schema) error                     { return nil }
func (c *dummyCache) DeletePluginConfig(_ *v1.PluginConfig) error         { return nil }
func (c *dummyCache) CheckServiceReference(_ *v1.Service) error           { return nil }
func (c *dummyCache) CheckPluginConfigReference(_ *v1.PluginConfig) error { return nil }
