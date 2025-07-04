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
)

type pluginMetadataClient struct {
	url     string
	cluster *cluster
}

func newPluginMetadataClient(c *cluster) *pluginMetadataClient {
	return &pluginMetadataClient{
		url:     c.baseURL + "/plugin_metadata",
		cluster: c,
	}
}

func (r *pluginMetadataClient) Get(ctx context.Context, name string) (*v1.PluginMetadata, error) {
	log.Debugw("try to look up pluginMetadata",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)

	// TODO Add mutex here to avoid dog-pile effect.
	url := r.url + "/" + name
	resp, err := r.cluster.getResource(ctx, url, "pluginMetadata")
	if err != nil {
		log.Errorw("failed to get pluginMetadata from APISIX",
			zap.String("name", name),
			zap.String("url", url),
			zap.String("cluster", r.cluster.name),
			zap.Error(err),
		)
		return nil, err
	}

	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		log.Errorw("failed to convert pluginMetadata item",
			zap.String("url", r.url),
			zap.Error(err),
		)
		return nil, err
	}
	return pluginMetadata, nil
}

func (r *pluginMetadataClient) List(ctx context.Context) (list []*v1.PluginMetadata, err error) {
	log.Debugw("try to list pluginMetadatas in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	var resp = struct {
		Value map[string]map[string]any
	}{}
	err = r.cluster.listResourceToResponse(ctx, r.url, "plugin_metadata", &resp)
	if err != nil {
		log.Errorf("failed to list pluginMetadatas: %s", err)
		return nil, err
	}
	for name, metadata := range resp.Value {
		list = append(list, &v1.PluginMetadata{
			Name:     name,
			Metadata: metadata,
		})
	}

	return
}

func (r *pluginMetadataClient) Delete(ctx context.Context, obj *v1.PluginMetadata) error {
	log.Debugw("try to delete pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.Name
	if err := r.cluster.deleteResource(ctx, url, "pluginMetadata"); err != nil {
		return err
	}
	return nil
}

func (r *pluginMetadataClient) Update(ctx context.Context, obj *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	url := r.url + "/" + obj.Name
	return updateResource(
		ctx,
		obj,
		url,
		"pluginMetadata",
		r.cluster.updateResource,
		func(obj *v1.PluginMetadata) error {
			return nil
		},
		func(resp *getResponse) (*v1.PluginMetadata, error) {
			return resp.pluginMetadata()
		},
	)
}

func (r *pluginMetadataClient) Create(ctx context.Context, obj *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	log.Debugw("try to create pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj.Metadata)
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.Name
	resp, err := r.cluster.updateResource(ctx, url, "pluginMetadata", body)
	if err != nil {
		return nil, err
	}
	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		return nil, err
	}
	return pluginMetadata, nil
}
