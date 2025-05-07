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

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"go.uber.org/zap"

	"github.com/api7/gopkg/pkg/log"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
	"github.com/api7/api7-ingress-controller/pkg/dashboard/cache"
)

type sslClient struct {
	url     string
	cluster *cluster
}

func newSSLClient(c *cluster) SSL {
	return &sslClient{
		url:     c.baseURL + "/ssls",
		cluster: c,
	}
}

// name is namespace_sslname
func (s *sslClient) Get(ctx context.Context, name string) (*v1.Ssl, error) {
	return getFromCacheOrAPI(
		ctx,
		id.GenID(name),
		s.url,
		s.cluster.cache.GetSSL,
		s.cluster.cache.InsertSSL,
		s.cluster.GetSSL,
	)
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (s *sslClient) List(ctx context.Context, listOptions ...any) ([]*v1.Ssl, error) {
	var options ListOptions
	if len(listOptions) > 0 {
		options = listOptions[0].(ListOptions)
	}
	if options.From == ListFromCache {
		log.Debugw("try to list ssls in cache",
			zap.String("cluster", s.cluster.name),
			zap.String("url", s.url),
		)
		return s.cluster.cache.ListSSL(
			"label",
			options.KindLabel.Kind,
			options.KindLabel.Namespace,
			options.KindLabel.Name,
		)
	}
	log.Debugw("try to list ssl in APISIX",
		zap.String("url", s.url),
		zap.String("cluster", s.cluster.name),
	)
	url := s.url
	sslItems, err := s.cluster.listResource(ctx, url, "ssls")
	if err != nil {
		log.Errorf("failed to list ssl: %s", err)
		return nil, err
	}

	items := make([]*v1.Ssl, 0, len(sslItems.List))
	for _, item := range sslItems.List {
		ssl, err := item.ssl()
		if err != nil {
			log.Errorw("failed to convert ssl item",
				zap.String("url", url),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, ssl)
	}

	return items, nil
}

func (s *sslClient) Create(ctx context.Context, obj *v1.Ssl) (*v1.Ssl, error) {
	log.Debugw("try to create ssl",
		zap.String("cluster", s.cluster.name),
		zap.String("url", s.url),
		zap.String("id", obj.ID),
	)
	if err := s.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := s.url + "/" + obj.ID
	log.Debugw("creating ssl", zap.ByteString("body", data), zap.String("url", url))
	resp, err := s.cluster.createResource(ctx, url, "ssls", data)
	if err != nil {
		log.Errorf("failed to create ssl: %s", err)
		return nil, err
	}

	ssl, err := resp.ssl()
	if err != nil {
		return nil, err
	}
	if err := s.cluster.cache.InsertSSL(ssl); err != nil {
		log.Errorf("failed to reflect ssl create to cache: %s", err)
		return nil, err
	}
	return ssl, nil
}

func (s *sslClient) Delete(ctx context.Context, obj *v1.Ssl) error {
	log.Debugw("try to delete ssl",
		zap.String("id", obj.ID),
		zap.String("cluster", s.cluster.name),
		zap.String("url", s.url),
	)
	if err := s.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := s.url + "/" + obj.ID
	if err := s.cluster.deleteResource(ctx, url, "ssls"); err != nil {
		return err
	}
	if err := s.cluster.cache.DeleteSSL(obj); err != nil {
		log.Errorf("failed to reflect ssl delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (s *sslClient) Update(ctx context.Context, obj *v1.Ssl) (*v1.Ssl, error) {
	url := s.url + "/" + obj.ID
	return updateResource(
		ctx,
		obj,
		url,
		"ssls",
		s.cluster.updateResource,
		s.cluster.cache.InsertSSL,
		func(resp *getResponse) (*v1.Ssl, error) {
			return resp.ssl()
		},
	)
}
