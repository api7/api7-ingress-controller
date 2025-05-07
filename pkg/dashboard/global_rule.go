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

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/api7/gopkg/pkg/log"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/pkg/dashboard/cache"
)

type globalRuleClient struct {
	url     string
	cluster *cluster
}

func newGlobalRuleClient(c *cluster) GlobalRule {
	return &globalRuleClient{
		url:     c.baseURL + "/global_rules",
		cluster: c,
	}
}

// Get returns the GlobalRule.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *globalRuleClient) Get(ctx context.Context, id string) (*v1.GlobalRule, error) {
	return getFromCacheOrAPI(
		ctx,
		id,
		r.url,
		r.cluster.cache.GetGlobalRule,
		r.cluster.cache.InsertGlobalRule,
		r.cluster.GetGlobalRule,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *globalRuleClient) List(ctx context.Context) ([]*v1.GlobalRule, error) {
	log.Debugw("try to list global_rules in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	url := r.url
	globalRuleItems, err := r.cluster.listResource(ctx, url, "globalRule")
	if err != nil {
		log.Errorf("failed to list global_rules: %s", err)
		return nil, err
	}

	items := make([]*v1.GlobalRule, 0, len(globalRuleItems.List))
	for _, item := range globalRuleItems.List {
		globalRule, err := item.globalRule()
		if err != nil {
			log.Errorw("failed to convert global_rule item",
				zap.String("url", r.url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, globalRule)
	}

	return items, nil
}

func (r *globalRuleClient) Create(ctx context.Context, obj *v1.GlobalRule) (*v1.GlobalRule, error) {
	// Overwrite global rule ID with the plugin name
	if len(obj.Plugins) == 0 { // This case will not happen as its handled at schema validation level
		return nil, fmt.Errorf("global rule must have at least one plugin")
	}

	// This is checked on dashboard that global rule id should be the plugin name
	for pluginName := range obj.Plugins {
		obj.ID = pluginName
		break
	}

	log.Debugw("try to create global_rule",
		zap.String("id", obj.ID),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.ID
	log.Debugw("creating global_rule", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, "globalRule", data)
	if err != nil {
		log.Errorf("failed to create global_rule: %s", err)
		return nil, err
	}

	globalRules, err := resp.globalRule()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertGlobalRule(globalRules); err != nil {
		log.Errorf("failed to reflect global_rules create to cache: %s", err)
		return nil, err
	}
	return globalRules, nil
}

func (r *globalRuleClient) Delete(ctx context.Context, obj *v1.GlobalRule) error {
	log.Debugw("try to delete global_rule",
		zap.String("id", obj.ID),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "globalRule"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *globalRuleClient) Update(ctx context.Context, obj *v1.GlobalRule) (*v1.GlobalRule, error) {
	url := r.url + "/" + obj.ID
	return updateResource(
		ctx,
		obj,
		url,
		"globalRule",
		r.cluster.updateResource,
		r.cluster.cache.InsertGlobalRule,
		func(gr *getResponse) (*v1.GlobalRule, error) {
			return gr.globalRule()
		},
	)
}
