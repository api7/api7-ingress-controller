// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/apache/apisix-ingress-controller/pkg/id"
	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/pkg/dashboard/cache"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
)

type consumerClient struct {
	url     string
	cluster *cluster
}

func newConsumerClient(c *cluster) Consumer {
	return &consumerClient{
		url:     c.baseURL + "/consumers",
		cluster: c,
	}
}

// Get returns the Consumer.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *consumerClient) Get(ctx context.Context, name string) (*v1.Consumer, error) {
	return getFromCacheOrAPI(
		ctx,
		id.GenID(name),
		r.url,
		r.cluster.cache.GetConsumer,
		r.cluster.cache.InsertConsumer,
		r.cluster.GetConsumer,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *consumerClient) List(ctx context.Context) ([]*v1.Consumer, error) {
	log.Debugw("try to list consumers in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	url := r.url
	consumerItems, err := r.cluster.listResource(ctx, url, "consumer")
	if err != nil {
		log.Errorf("failed to list consumers: %s", err)
		return nil, err
	}
	items := make([]*v1.Consumer, 0, len(consumerItems.List))
	for _, item := range consumerItems.List {
		consumer, err := item.consumer()
		if err != nil {
			log.Errorw("failed to convert consumer item",
				zap.String("url", url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, consumer)
	}

	return items, nil
}

func (r *consumerClient) Create(ctx context.Context, obj *v1.Consumer) (*v1.Consumer, error) {
	log.Debugw("try to create consumer",
		zap.String("name", obj.Username),
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

	url := r.url + "/" + obj.Username
	resp, err := r.cluster.createResource(ctx, url, "consumer", data)
	if err != nil {
		log.Errorf("failed to create consumer: %s", err)
		return nil, err
	}
	consumer, err := resp.consumer()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertConsumer(consumer); err != nil {
		log.Errorf("failed to reflect consumer create to cache: %s", err)
		return nil, err
	}
	return consumer, nil
}

func (r *consumerClient) Delete(ctx context.Context, obj *v1.Consumer) error {
	log.Debugw("try to delete consumer",
		zap.String("name", obj.Username),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.Username
	if err := r.cluster.deleteResource(ctx, url, "consumer"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteConsumer(obj); err != nil {
		log.Errorf("failed to reflect consumer delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *consumerClient) Update(ctx context.Context, obj *v1.Consumer) (*v1.Consumer, error) {
	url := r.url + "/" + obj.Username
	return updateResource(
		ctx,
		obj,
		url,
		"consumer",
		r.cluster.updateResource,
		r.cluster.cache.InsertConsumer,
		func(resp *getResponse) (*v1.Consumer, error) {
			return resp.consumer()
		},
	)
}
