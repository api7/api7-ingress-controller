// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
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

	"github.com/apache/apisix-ingress-controller/pkg/id"
	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/pkg/dashboard/cache"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
)

type routeClient struct {
	url     string
	cluster *cluster
}

func newRouteClient(c *cluster) Route {
	return &routeClient{
		url:     c.baseURL + "/routes",
		cluster: c,
	}
}

// Get returns the Route.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *routeClient) Get(ctx context.Context, name string) (*v1.Route, error) {
	return getFromCacheOrAPI(
		ctx,
		id.GenID(name),
		r.url,
		r.cluster.cache.GetRoute,
		r.cluster.cache.InsertRoute,
		r.cluster.GetRoute,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *routeClient) List(ctx context.Context, args ...any) ([]*v1.Route, error) {
	log.Debugw("try to list routes in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	routeItems, err := r.cluster.listResource(ctx, r.url, "route")
	if err != nil {
		log.Errorf("failed to list routes: %s", err)
		return nil, err
	}

	items := make([]*v1.Route, 0, len(routeItems.List))
	for _, item := range routeItems.List {
		route, err := item.route()
		if err != nil {
			log.Errorw("failed to convert route item",
				zap.String("url", r.url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, route)
	}

	return items, nil
}

func (r *routeClient) Create(ctx context.Context, obj *v1.Route) (*v1.Route, error) {
	obj.Name = obj.ID
	log.Debugw("try to create route",
		zap.Strings("hosts", obj.Hosts),
		zap.String("name", obj.Name),
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
	resp, err := r.cluster.createResource(ctx, url, "route", data)
	if err != nil {
		log.Errorf("failed to create route: %s", err)
		return nil, err
	}

	route, err := resp.route()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	return route, nil
}

func (r *routeClient) Delete(ctx context.Context, obj *v1.Route) error {
	log.Debugw("try to delete route",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "route"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteRoute(obj); err != nil {
		log.Errorf("failed to reflect route delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *routeClient) Update(ctx context.Context, obj *v1.Route) (*v1.Route, error) {
	url := r.url + "/" + obj.ID
	return updateResource(
		ctx,
		obj,
		url,
		"route",
		r.cluster.updateResource,
		r.cluster.cache.InsertRoute,
		func(resp *getResponse) (*v1.Route, error) {
			return resp.route()
		},
	)
}
