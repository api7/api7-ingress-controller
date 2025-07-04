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
	"encoding/json"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
)

type pluginConfigClient struct {
	url     string
	cluster *cluster
}

func newPluginConfigClient(c *cluster) PluginConfig {
	return &pluginConfigClient{
		url:     c.baseURL + "/plugin_configs",
		cluster: c,
	}
}

// Get returns the v1.PluginConfig.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (pc *pluginConfigClient) Get(ctx context.Context, name string) (*v1.PluginConfig, error) {
	return getFromCacheOrAPI(
		ctx,
		id.GenID(name),
		pc.url,
		pc.cluster.cache.GetPluginConfig,
		pc.cluster.cache.InsertPluginConfig,
		pc.cluster.GetPluginConfig,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (pc *pluginConfigClient) List(ctx context.Context) ([]*v1.PluginConfig, error) {
	log.Debugw("try to list pluginConfig in APISIX",
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)
	pluginConfigItems, err := pc.cluster.listResource(ctx, pc.url, "pluginConfig")
	if err != nil {
		log.Errorf("failed to list pluginConfig: %s", err)
		return nil, err
	}

	items := make([]*v1.PluginConfig, 0, len(pluginConfigItems.List))
	for _, item := range pluginConfigItems.List {
		pluginConfig, err := item.pluginConfig()
		if err != nil {
			log.Errorw("failed to convert pluginConfig item",
				zap.String("url", pc.url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, pluginConfig)
	}

	return items, nil
}

func (pc *pluginConfigClient) Create(ctx context.Context, obj *v1.PluginConfig) (*v1.PluginConfig, error) {
	log.Debugw("try to create pluginConfig",
		zap.String("name", obj.Name),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)

	if err := pc.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := pc.url + "/" + obj.ID
	log.Debugw("creating pluginConfig", zap.ByteString("body", data), zap.String("url", url))
	resp, err := pc.cluster.createResource(ctx, url, "pluginConfig", data)
	if err != nil {
		log.Errorf("failed to create pluginConfig: %s", err)
		return nil, err
	}

	pluginConfig, err := resp.pluginConfig()
	if err != nil {
		return nil, err
	}
	if err := pc.cluster.cache.InsertPluginConfig(pluginConfig); err != nil {
		log.Errorf("failed to reflect pluginConfig create to cache: %s", err)
		return nil, err
	}
	return pluginConfig, nil
}

func (pc *pluginConfigClient) Delete(ctx context.Context, obj *v1.PluginConfig) error {
	log.Debugw("try to delete pluginConfig",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)
	err := pc.cluster.cache.CheckPluginConfigReference(obj)
	if err != nil {
		log.Warnw("deletion for plugin config: " + obj.Name + " aborted as it is still in use.")
		return err
	}
	if err := pc.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := pc.url + "/" + obj.ID
	if err := pc.cluster.deleteResource(ctx, url, "pluginConfig"); err != nil {
		return err
	}
	if err := pc.cluster.cache.DeletePluginConfig(obj); err != nil {
		log.Errorf("failed to reflect pluginConfig delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (pc *pluginConfigClient) Update(ctx context.Context, obj *v1.PluginConfig) (*v1.PluginConfig, error) {
	url := pc.url + "/" + obj.ID
	return updateResource(
		ctx,
		obj,
		url,
		"pluginConfig",
		pc.cluster.updateResource,
		pc.cluster.cache.InsertPluginConfig,
		func(resp *getResponse) (*v1.PluginConfig, error) {
			return resp.pluginConfig()
		},
	)
}
