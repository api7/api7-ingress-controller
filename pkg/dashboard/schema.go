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
	"strings"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
	"github.com/apache/apisix-ingress-controller/pkg/dashboard/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
)

type schemaClient struct {
	url     string
	cluster *cluster
}

func newSchemaClient(c *cluster) Schema {
	return &schemaClient{
		url:     c.baseURL + "/schema",
		cluster: c,
	}
}

// GetSchema returns APISIX object's schema.
func (sc schemaClient) getSchema(ctx context.Context, name string) (*v1.Schema, error) {
	log.Debugw("try to look up schema",
		zap.String("name", name),
		zap.String("url", sc.url),
		zap.String("cluster", sc.cluster.name),
	)

	sid := id.GenID(name)
	schema, err := sc.cluster.cache.GetSchema(sid)
	if err == nil {
		return schema, nil
	}
	if err == cache.ErrNotFound {
		log.Debugw("failed to find schema in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Errorw("failed to find schema in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}
	// Dashboard uses /apisix/admin/plugins/{plugin_name} instead of /apisix/admin/schema/{plugin_name} to get schema
	url := strings.Replace(sc.url, "schema", name, 1)
	content, err := sc.cluster.getSchema(ctx, url, "schema")
	if err != nil {
		log.Errorw("failed to get schema from APISIX",
			zap.String("name", name),
			zap.String("url", url),
			zap.String("cluster", sc.cluster.name),
			zap.Error(err),
		)
		return nil, err
	}
	schema = &v1.Schema{
		Name:    name,
		Content: content,
	}
	if err := sc.cluster.cache.InsertSchema(schema); err != nil {
		log.Errorf("failed to reflect schema create to cache: %s", err)
		return nil, err
	}
	return schema, nil
}

// GetPluginSchema returns plugin's schema.
func (sc schemaClient) GetPluginSchema(ctx context.Context, pluginName string) (*v1.Schema, error) {
	return sc.getSchema(ctx, "plugins/"+pluginName)
}

// GetRouteSchema returns route's schema.
func (sc schemaClient) GetRouteSchema(ctx context.Context) (*v1.Schema, error) {
	return sc.getSchema(ctx, "route")
}

// GetUpstreamSchema returns upstream's schema.
func (sc schemaClient) GetUpstreamSchema(ctx context.Context) (*v1.Schema, error) {
	return sc.getSchema(ctx, "upstream")
}

// GetConsumerSchema returns consumer's schema.
func (sc schemaClient) GetConsumerSchema(ctx context.Context) (*v1.Schema, error) {
	return sc.getSchema(ctx, "consumer")
}

// GetSslSchema returns SSL's schema.
func (sc schemaClient) GetSslSchema(ctx context.Context) (*v1.Schema, error) {
	return sc.getSchema(ctx, "ssl")
}

// GetPluginConfigSchema returns PluginConfig's schema.
func (sc schemaClient) GetPluginConfigSchema(ctx context.Context) (*v1.Schema, error) {
	return sc.getSchema(ctx, "pluginConfig")
}
