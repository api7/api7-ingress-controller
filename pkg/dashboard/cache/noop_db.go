// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

type noopCache struct {
}

// NewMemDBCache creates a Cache object backs with a memory DB.
func NewNoopDBCache() (Cache, error) {
	return &noopCache{}, nil
}

func (c *noopCache) InsertRoute(r *v1.Route) error {
	return nil
}

func (c *noopCache) InsertSSL(ssl *v1.Ssl) error {
	return nil
}

func (c *noopCache) InsertService(u *v1.Service) error {
	return nil
}

func (c *noopCache) InsertStreamRoute(sr *v1.StreamRoute) error {
	return nil
}

func (c *noopCache) InsertGlobalRule(gr *v1.GlobalRule) error {
	return nil
}

func (c *noopCache) InsertConsumer(consumer *v1.Consumer) error {
	return nil
}

func (c *noopCache) InsertSchema(schema *v1.Schema) error {
	return nil
}

func (c *noopCache) InsertPluginConfig(pc *v1.PluginConfig) error {
	return nil
}

func (c *noopCache) GetRoute(id string) (*v1.Route, error) {
	return nil, nil
}

func (c *noopCache) GetSSL(id string) (*v1.Ssl, error) {
	return nil, nil
}

func (c *noopCache) GetService(id string) (*v1.Service, error) {
	return nil, nil
}

func (c *noopCache) GetStreamRoute(id string) (*v1.StreamRoute, error) {
	return nil, nil
}

func (c *noopCache) GetGlobalRule(id string) (*v1.GlobalRule, error) {
	return nil, nil
}

func (c *noopCache) GetConsumer(username string) (*v1.Consumer, error) {
	return nil, nil
}

func (c *noopCache) GetSchema(name string) (*v1.Schema, error) {
	return nil, nil
}

func (c *noopCache) GetPluginConfig(name string) (*v1.PluginConfig, error) {
	return nil, nil
}

func (c *noopCache) ListRoutes(...any) ([]*v1.Route, error) {
	return nil, nil
}

func (c *noopCache) ListSSL(...any) ([]*v1.Ssl, error) {
	return nil, nil
}

func (c *noopCache) ListServices(...any) ([]*v1.Service, error) {
	return nil, nil
}

func (c *noopCache) ListStreamRoutes() ([]*v1.StreamRoute, error) {
	return nil, nil
}

func (c *noopCache) ListGlobalRules() ([]*v1.GlobalRule, error) {
	return nil, nil
}

func (c *noopCache) ListConsumers() ([]*v1.Consumer, error) {
	return nil, nil
}

func (c *noopCache) ListSchema() ([]*v1.Schema, error) {
	return nil, nil
}

func (c *noopCache) ListPluginConfigs() ([]*v1.PluginConfig, error) {
	return nil, nil
}

func (c *noopCache) DeleteRoute(r *v1.Route) error {
	return nil
}

func (c *noopCache) DeleteSSL(ssl *v1.Ssl) error {
	return nil
}

func (c *noopCache) DeleteService(u *v1.Service) error {
	return nil
}

func (c *noopCache) DeleteStreamRoute(sr *v1.StreamRoute) error {
	return nil
}

func (c *noopCache) DeleteGlobalRule(gr *v1.GlobalRule) error {
	return nil
}

func (c *noopCache) DeleteConsumer(consumer *v1.Consumer) error {
	return nil
}

func (c *noopCache) DeleteSchema(schema *v1.Schema) error {
	return nil
}

func (c *noopCache) DeletePluginConfig(pc *v1.PluginConfig) error {
	return nil
}

func (c *noopCache) CheckServiceReference(u *v1.Service) error {
	return nil
}

func (c *noopCache) CheckPluginConfigReference(pc *v1.PluginConfig) error {
	return nil
}
