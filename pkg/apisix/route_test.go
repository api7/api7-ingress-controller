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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"

	"github.com/api7/api7-ingress-controller/pkg/metrics"
	v1 "github.com/api7/api7-ingress-controller/pkg/types/apisix/v1"
)

type fakeAPISIXRouteSrv struct {
	route map[string]map[string]interface{}
}

func (srv *fakeAPISIXRouteSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/routes") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		//For individual resource, the getcreate response is sent
		var key string
		if strings.HasPrefix(r.URL.Path, "/apisix/admin/routes/") && strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/") != "" {
			key = strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
		}
		if key != "" {
			resp := fakeGetCreateResp{
				fakeGetCreateItem{
					Key:   key,
					Value: srv.route[key],
				},
			}
			resp.fakeGetCreateItem.Value = srv.route[key]
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
		} else {
			resp := fakeListResp{}
			resp.Total = fmt.Sprintf("%d", len(srv.route))
			resp.List = make([]fakeListItem, 0, len(srv.route))
			for _, v := range srv.route {
				resp.List = append(resp.List, v)
			}
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
		}

		return
	}

	if r.Method == http.MethodDelete {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
		id = "/apisix/admin/routes/" + id
		code := http.StatusNotFound
		if _, ok := srv.route[id]; ok {
			delete(srv.route, id)
			code = http.StatusOK
		}
		w.WriteHeader(code)
	}

	if r.Method == http.MethodPut {
		paths := strings.Split(r.URL.Path, "/")
		key := fmt.Sprintf("/apisix/admin/routes/%s", paths[len(paths)-1])
		data, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		gr := make(map[string]interface{}, 0)
		json.Unmarshal(data, &gr)
		srv.route[key] = gr
		fmt.Println("BHAI  add kiya with key", key, "=", srv.route[key])
		var val Value
		json.Unmarshal(data, &val)
		resp := fakeGetCreateResp{
			fakeGetCreateItem{
				Value: val,
				Key:   key,
			},
		}
		data, _ = json.Marshal(resp)
		_, _ = w.Write(data)
		return
	}

	if r.Method == http.MethodPatch {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
		id = "/apisix/routes/" + id
		if _, ok := srv.route[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data, _ := io.ReadAll(r.Body)
		var val Value
		json.Unmarshal(data, &val)
		gr := make(map[string]interface{}, 0)
		json.Unmarshal(data, &gr)
		srv.route[id] = gr
		w.WriteHeader(http.StatusOK)
		resp := fakeGetCreateResp{
			fakeGetCreateItem{
				Value: val,
				Key:   id,
			},
		}
		byt, _ := json.Marshal(resp)
		_, _ = w.Write([]byte(byt))
		return
	}
}

func runFakeRouteSrv(t *testing.T) *http.Server {
	srv := &fakeAPISIXRouteSrv{
		route: make(map[string]map[string]interface{}),
	}

	ln, _ := nettest.NewLocalListener("tcp")

	httpSrv := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: srv,
	}

	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Errorf("failed to run http server: %s", err)
		}
	}()

	return httpSrv
}

func TestRouteClient(t *testing.T) {
	srv := runFakeRouteSrv(t)
	defer func() {
		assert.Nil(t, srv.Shutdown(context.Background()))
	}()

	u := url.URL{
		Scheme: "http",
		Host:   srv.Addr,
		Path:   "/apisix/admin",
	}

	closedCh := make(chan struct{})
	close(closedCh)
	cli := newRouteClient(&cluster{
		baseURL:           u.String(),
		cli:               http.DefaultClient,
		cache:             &dummyCache{},
		generatedObjCache: &dummyCache{},
		cacheSynced:       closedCh,
		metricsCollector:  metrics.NewPrometheusCollector(),
	})

	// Create
	obj, err := cli.Create(context.Background(), &v1.Route{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "test",
		},
		Host:      "www.foo.com",
		Uri:       "/bar",
		ServiceID: "1",
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, "1", obj.ID)

	obj, err = cli.Create(context.Background(), &v1.Route{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: "test",
		},
		Host:      "www.foo.com",
		Uri:       "/bar",
		ServiceID: "1",
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, "2", obj.ID)

	// List
	objs, err := cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.Equal(t, "1", objs[0].ID)
	assert.Equal(t, "2", objs[1].ID)

	// Delete then List
	assert.Nil(t, cli.Delete(context.Background(), objs[0]))
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].ID)

	// Patch then List
	_, err = cli.Update(context.Background(), &v1.Route{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: "test",
		},
		Host:      "www.foo.com",
		Uri:       "/bar",
		ServiceID: "112",
	}, false)
	assert.Nil(t, err)
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].ID)
}
