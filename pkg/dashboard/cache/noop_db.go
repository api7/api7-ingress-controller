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

func (c *noopCache) ListRoutes(...interface{}) ([]*v1.Route, error) {
	return nil, nil
}

func (c *noopCache) ListSSL() ([]*v1.Ssl, error) {
	return nil, nil
}

func (c *noopCache) ListServices(...interface{}) ([]*v1.Service, error) {
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
