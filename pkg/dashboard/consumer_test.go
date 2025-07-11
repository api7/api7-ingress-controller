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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
)

type fakeAPISIXConsumerSrv struct {
	consumer map[string]map[string]any
}

type Value map[string]any

type fakeListResp struct {
	Total string         `json:"total"`
	List  []fakeListItem `json:"list"`
}

type fakeGetCreateResp struct {
	fakeGetCreateItem
}

type fakeGetCreateItem struct {
	Value Value  `json:"value"`
	Key   string `json:"key"`
}

type fakeListItem Value

func (srv *fakeAPISIXConsumerSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/consumers") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		// For individual resource, the getcreate response is sent
		var key string
		if strings.HasPrefix(r.URL.Path, "/apisix/admin/consumers/") &&
			strings.TrimPrefix(r.URL.Path, "/apisix/admin/consumers/") != "" {
			key = strings.TrimPrefix(r.URL.Path, "/apisix/admin/consumers/")
		}
		if key != "" {
			resp := fakeGetCreateResp{
				fakeGetCreateItem{
					Key:   key,
					Value: srv.consumer[key],
				},
			}
			resp.Value = srv.consumer[key]
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
		} else {
			resp := fakeListResp{}
			resp.Total = fmt.Sprintf("%d", len(srv.consumer))
			resp.List = make([]fakeListItem, 0, len(srv.consumer))
			for _, v := range srv.consumer {
				resp.List = append(resp.List, v)
			}
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
		}

		return
	}

	if r.Method == http.MethodDelete {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/consumers/")
		id = "/apisix/admin/consumers/" + id
		code := http.StatusNotFound
		if _, ok := srv.consumer[id]; ok {
			delete(srv.consumer, id)
			code = http.StatusOK
		}
		w.WriteHeader(code)
	}

	if r.Method == http.MethodPut {
		paths := strings.Split(r.URL.Path, "/")
		key := fmt.Sprintf("/apisix/admin/consumers/%s", paths[len(paths)-1])
		data, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		consumer := make(map[string]any, 0)
		_ = json.Unmarshal(data, &consumer)
		srv.consumer[key] = consumer
		var val Value
		_ = json.Unmarshal(data, &val)
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
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/consumers/")
		id = "/apisix/admin/consumers/" + id
		if _, ok := srv.consumer[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data, _ := io.ReadAll(r.Body)
		var val Value
		_ = json.Unmarshal(data, &val)
		consumer := make(map[string]any, 0)
		_ = json.Unmarshal(data, &consumer)
		srv.consumer[id] = consumer
		w.WriteHeader(http.StatusOK)
		resp := fakeGetCreateResp{
			fakeGetCreateItem{
				Value: val,
				Key:   id,
			},
		}
		byt, _ := json.Marshal(resp)
		_, _ = w.Write(byt)
		return
	}
}

func runFakeConsumerSrv(t *testing.T) *http.Server {
	srv := &fakeAPISIXConsumerSrv{
		consumer: make(map[string]map[string]any),
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

func TestConsumerClient(t *testing.T) {
	srv := runFakeConsumerSrv(t)
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
	cli := newConsumerClient(&cluster{
		baseURL:     u.String(),
		cli:         http.DefaultClient,
		cache:       &dummyCache{},
		cacheSynced: closedCh,
	})

	// Create
	obj, err := cli.Create(context.Background(), &v1.Consumer{
		Username: "1",
	})
	assert.Nil(t, err)
	assert.Equal(t, "1", obj.Username)

	obj, err = cli.Create(context.Background(), &v1.Consumer{
		Username: "2",
	})
	assert.Nil(t, err)
	assert.Equal(t, "2", obj.Username)

	// List
	objs, err := cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.ElementsMatch(t, []string{"1", "2"}, []string{objs[0].Username, objs[1].Username})

	// Delete then List
	if objs[0].Username != "1" {
		objs[0], objs[1] = objs[1], objs[0]
	}
	assert.Nil(t, cli.Delete(context.Background(), objs[0]))
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].Username)

	// Patch then List
	_, err = cli.Update(context.Background(), &v1.Consumer{
		Username: "2",
		Plugins: map[string]any{
			"prometheus": struct{}{},
		},
	})
	assert.Nil(t, err)
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].Username)
}
