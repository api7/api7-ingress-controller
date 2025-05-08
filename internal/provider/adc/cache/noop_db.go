package cache

import (
	types "github.com/api7/api7-ingress-controller/api/adc"
)

type noopCache struct {
}

// NewMemDBCache creates a Cache object backs with a memory DB.
func NewNoopDBCache() (Cache, error) {
	return &noopCache{}, nil
}

func (c *noopCache) Insert(obj any) error {
	return nil
}

func (c *noopCache) Delete(obj any) error {
	return nil
}

func (c *noopCache) InsertSSL(ssl *types.SSL) error {
	return nil
}

func (c *noopCache) InsertService(u *types.Service) error {
	return nil
}

func (c *noopCache) InsertGlobalRule(gr *types.GlobalRule) error {
	return nil
}

func (c *noopCache) InsertConsumer(consumer *types.Consumer) error {
	return nil
}

func (c *noopCache) GetSSL(id string) (*types.SSL, error) {
	return nil, nil
}

func (c *noopCache) GetService(id string) (*types.Service, error) {
	return nil, nil
}

func (c *noopCache) GetGlobalRule(id string) (*types.GlobalRule, error) {
	return nil, nil
}

func (c *noopCache) GetConsumer(username string) (*types.Consumer, error) {
	return nil, nil
}

func (c *noopCache) ListSSL(...ListOption) ([]*types.SSL, error) {
	return nil, nil
}

func (c *noopCache) ListServices(...ListOption) ([]*types.Service, error) {
	return nil, nil
}

func (c *noopCache) ListStreamRoutes(...ListOption) ([]*types.StreamRoute, error) {
	return nil, nil
}

func (c *noopCache) ListGlobalRules(...ListOption) ([]*types.GlobalRule, error) {
	return nil, nil
}

func (c *noopCache) ListConsumers(...ListOption) ([]*types.Consumer, error) {
	return nil, nil
}

func (c *noopCache) DeleteSSL(ssl *types.SSL) error {
	return nil
}

func (c *noopCache) DeleteService(u *types.Service) error {
	return nil
}

func (c *noopCache) DeleteGlobalRule(gr *types.GlobalRule) error {
	return nil
}

func (c *noopCache) DeleteConsumer(consumer *types.Consumer) error {
	return nil
}
