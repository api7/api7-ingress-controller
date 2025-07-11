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

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
)

type pluginClient struct {
	url     string
	cluster *cluster
}

func newPluginClient(c *cluster) Plugin {
	return &pluginClient{
		url:     c.baseURL + "/plugins",
		cluster: c,
	}
}

// List returns the names of all plugins.
func (p *pluginClient) List(ctx context.Context) ([]string, error) {
	log.Debugw("try to list plugin names in APISIX",
		zap.String("cluster", p.cluster.name),
		zap.String("url", p.url),
	)
	url := p.url + "/list"
	pluginList, err := p.cluster.getList(ctx, url, "plugin")
	if err != nil {
		log.Errorf("failed to list plugin names: %s", err)
		return nil, err
	}
	log.Debugf("plugin list: %v", pluginList)
	return pluginList, nil
}
