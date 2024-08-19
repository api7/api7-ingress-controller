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
package apisix

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/api7/api7-ingress/api/apisix/v1"
	"github.com/api7/api7-ingress/pkg/apisix/cache"
)

type serviceClient struct {
	url     string
	cluster *cluster
}

func newServiceClient(c *cluster) Service {
	return &serviceClient{
		url:     c.baseURL + "/services",
		cluster: c,
	}
}

func (u *serviceClient) Get(ctx context.Context, name string) (*v1.Service, error) {
	log.Debugw("try to look up upstream",
		zap.String("name", name),
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)
	uid := id.GenID(name)
	svc, err := u.cluster.cache.GetService(uid)
	if err == nil {
		return svc, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect
	svc, err = u.cluster.GetService(ctx, u.url, uid)
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertService(svc); err != nil {
		log.Errorf("failed to reflect upstream create to cache: %s", err)
		return nil, err
	}
	return svc, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (u *serviceClient) List(ctx context.Context) ([]*v1.Service, error) {
	log.Debugw("try to list upstreams in APISIX",
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)
	upsItems, err := u.cluster.listResource(ctx, u.url, "service")
	if err != nil {
		log.Errorf("failed to list upstreams: %s", err)
		return nil, err
	}

	var items []*v1.Service
	for _, item := range upsItems.List {
		ups, err := item.service()
		if err != nil {
			log.Errorw("failed to convert upstream item",
				zap.String("url", u.url),
				zap.Error(err),
			)
			return nil, err
		}
		items = append(items, ups)
	}
	return items, nil
}

func (u *serviceClient) Create(ctx context.Context, obj *v1.Service) (*v1.Service, error) {
	log.Debugw("try to create upstream",
		zap.String("name", obj.Name),
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	serviceObj := *obj
	body, err := json.Marshal(serviceObj)
	if err != nil {
		return nil, err
	}
	url := u.url + "/" + obj.ID
	log.Debugw("creating service", zap.ByteString("body", body), zap.String("url", url))
	resp, err := u.cluster.createResource(ctx, url, "service", body)
	if err != nil {
		log.Errorf("failed to create upstream: %s", err)
		return nil, err
	}
	ups, err := resp.service()
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertService(ups); err != nil {
		log.Errorf("failed to reflect upstream create to cache: %s", err)
		return nil, err
	}
	return ups, err
}

func (u *serviceClient) Delete(ctx context.Context, obj *v1.Service) error {
	log.Debugw("try to delete upstream",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", u.cluster.name),
		zap.String("url", u.url),
	)
	err := u.cluster.cache.CheckServiceReference(obj)
	if err != nil {
		log.Warnw("deletion for upstream: " + obj.Name + " aborted as it is still in use.")
		return err
	}
	if err := u.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := u.url + "/" + obj.ID
	if err := u.cluster.deleteResource(ctx, url, "service"); err != nil {
		return err
	}
	if err := u.cluster.cache.DeleteService(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to cache: %s", err.Error())
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (u *serviceClient) Update(ctx context.Context, obj *v1.Service) (*v1.Service, error) {
	log.Debugw("try to update upstream",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", u.cluster.name),
		zap.String("url", u.url),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}

	svcObj := *obj
	body, err := json.Marshal(svcObj)
	if err != nil {
		return nil, err
	}

	url := u.url + "/" + obj.ID
	resp, err := u.cluster.updateResource(ctx, url, "service", body)
	if err != nil {
		return nil, err
	}
	svc, err := resp.service()
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertService(svc); err != nil {
		log.Errorf("failed to reflect upstream update to cache: %s", err)
		return nil, err
	}
	return svc, err
}
