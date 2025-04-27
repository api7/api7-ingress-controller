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

type streamRouteClient struct {
	url     string
	cluster *cluster
}

func newStreamRouteClient(c *cluster) StreamRoute {
	url := c.baseURL + "/stream_routes"
	_, err := c.listResource(context.Background(), url, "streamRoute")
	if err == ErrFunctionDisabled {
		log.Infow("resource stream_routes is disabled")
		return &noopClient{}
	}
	return &streamRouteClient{
		url:     url,
		cluster: c,
	}
}

// Get returns the StreamRoute.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *streamRouteClient) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	return getFromCacheOrAPI(
		ctx,
		id.GenID(name),
		r.url,
		r.cluster.cache.GetStreamRoute,
		r.cluster.cache.InsertStreamRoute,
		r.cluster.GetStreamRoute,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *streamRouteClient) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	log.Debugw("try to list stream_routes in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	streamRouteItems, err := r.cluster.listResource(ctx, r.url, "streamRoute")
	if err != nil {
		log.Errorf("failed to list stream_routes: %s", err)
		return nil, err
	}

	items := make([]*v1.StreamRoute, 0, len(streamRouteItems.List))
	for _, item := range streamRouteItems.List {
		streamRoute, err := item.streamRoute()
		if err != nil {
			log.Errorw("failed to convert stream_route item",
				zap.String("url", r.url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, streamRoute)
	}
	return items, nil
}

func (r *streamRouteClient) Create(ctx context.Context, obj *v1.StreamRoute) (*v1.StreamRoute, error) {
	log.Debugw("try to create stream_route",
		zap.String("id", obj.ID),
		zap.Int32("server_port", obj.ServerPort),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
		zap.String("sni", obj.SNI),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.ID
	log.Infow("creating stream_route", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, "streamRoute", data)
	if err != nil {
		log.Errorf("failed to create stream_route: %s", err)
		return nil, err
	}

	streamRoute, err := resp.streamRoute()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertStreamRoute(streamRoute); err != nil {
		log.Errorf("failed to reflect stream_route create to cache: %s", err)
		return nil, err
	}
	return streamRoute, nil
}

func (r *streamRouteClient) Delete(ctx context.Context, obj *v1.StreamRoute) error {
	log.Debugw("try to delete stream_route",
		zap.String("id", obj.ID),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "streamRoute"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *streamRouteClient) Update(ctx context.Context, obj *v1.StreamRoute) (*v1.StreamRoute, error) {
	url := r.url + "/" + obj.ID
	return updateResource(
		ctx,
		obj,
		url,
		"streamRoute",
		r.cluster.updateResource,
		r.cluster.cache.InsertStreamRoute,
		func(resp *getResponse) (*v1.StreamRoute, error) {
			return resp.streamRoute()
		},
	)
}
