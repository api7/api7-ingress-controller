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

package apisix

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"net/url"
// 	"strings"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"golang.org/x/net/nettest"

// 	"github.com/api7/api7-ingress-controller/pkg/metrics"
// 	v1 "github.com/api7/api7-ingress-controller/pkg/types/apisix/v1"
// )

// type fakeAPISIXSSLSrv struct {
// 	ssl map[string]json.RawMessage
// }

// func (srv *fakeAPISIXSSLSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	defer r.Body.Close()

// 	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/ssl") {
// 		w.WriteHeader(http.StatusNotFound)
// 		return
// 	}

// 	if r.Method == http.MethodGet {
// 		//For individual resource, the getcreate response is sent
// 		key := strings.TrimPrefix(r.URL.Path, "/apisix/admin/ssl/")
// 		if key != "" {
// 			resp := fakeGetCreateResp{}
// 			resp.fakeGetCreateItem.Value = srv.ssl[key]
// 			w.WriteHeader(http.StatusOK)
// 			data, _ := json.Marshal(resp)
// 			_, _ = w.Write(data)
// 		} else {
// 			resp := fakeListResp{}
// 			resp.Total = fmt.Sprintf("%d", len(srv.ssl))
// 			resp.List = make([]fakeListItem, 0, len(srv.ssl))
// 			for _, v := range srv.ssl {
// 				resp.List = append(resp.List, fakeListItem(v))
// 			}
// 		}

// 		return
// 	}

// 	if r.Method == http.MethodDelete {
// 		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/ssl/")
// 		id = "/apisix/ssl/" + id
// 		code := http.StatusNotFound
// 		if _, ok := srv.ssl[id]; ok {
// 			delete(srv.ssl, id)
// 			code = http.StatusOK
// 		}
// 		w.WriteHeader(code)
// 	}

// 	if r.Method == http.MethodPut {
// 		paths := strings.Split(r.URL.Path, "/")
// 		key := fmt.Sprintf("/apisix/admin/ssl/%s", paths[len(paths)-1])
// 		data, _ := io.ReadAll(r.Body)
// 		srv.ssl[key] = data
// 		w.WriteHeader(http.StatusCreated)
// 		resp := fakeGetCreateResp{
// 			fakeGetCreateItem{
// 				Value: data,
// 			},
// 		}
// 		data, _ = json.Marshal(resp)
// 		_, _ = w.Write(data)
// 		return
// 	}

// 	if r.Method == http.MethodPatch {
// 		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/ssl/")
// 		id = "/apisix/ssl/" + id
// 		if _, ok := srv.ssl[id]; !ok {
// 			w.WriteHeader(http.StatusNotFound)
// 			return
// 		}

// 		data, _ := io.ReadAll(r.Body)
// 		srv.ssl[id] = data

// 		w.WriteHeader(http.StatusOK)
// 		resp := fakeGetCreateResp{
// 			fakeGetCreateItem{
// 				Value: data,
// 			},
// 		}
// 		byt, _ := json.Marshal(resp)
// 		_, _ = w.Write([]byte(byt))
// 		return
// 	}
// }

// func runFakeSSLSrv(t *testing.T) *http.Server {
// 	srv := &fakeAPISIXSSLSrv{
// 		ssl: make(map[string]json.RawMessage),
// 	}

// 	ln, _ := nettest.NewLocalListener("tcp")
// 	httpSrv := &http.Server{
// 		Addr:    ln.Addr().String(),
// 		Handler: srv,
// 	}

// 	go func() {
// 		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
// 			t.Errorf("failed to run http server: %s", err)
// 		}
// 	}()

// 	return httpSrv
// }

// func TestSSLClient(t *testing.T) {
// 	srv := runFakeSSLSrv(t)
// 	defer func() {
// 		assert.Nil(t, srv.Shutdown(context.Background()))
// 	}()

// 	u := url.URL{
// 		Scheme: "http",
// 		Host:   srv.Addr,
// 		Path:   "/apisix/admin",
// 	}
// 	closedCh := make(chan struct{})
// 	close(closedCh)

// 	cli := newSSLClient(&cluster{
// 		baseURL:           u.String(),
// 		cli:               http.DefaultClient,
// 		cache:             &dummyCache{},
// 		generatedObjCache: &dummyCache{},
// 		cacheSynced:       closedCh,
// 		metricsCollector:  metrics.NewPrometheusCollector(),
// 	})

// 	// Create
// 	obj, err := cli.Create(context.TODO(), &v1.Ssl{
// 		ID:   "1",
// 		Snis: []string{"bar.com"},
// 	}, false)
// 	assert.Nil(t, err)
// 	assert.Equal(t, "1", obj.ID)

// 	obj, err = cli.Create(context.TODO(), &v1.Ssl{
// 		ID:   "2",
// 		Snis: []string{"bar.com"},
// 	}, false)
// 	assert.Nil(t, err)
// 	assert.Equal(t, "2", obj.ID)

// 	// List
// 	objs, err := cli.List(context.Background())
// 	assert.Nil(t, err)
// 	assert.Len(t, objs, 2)
// 	assert.Equal(t, "1", objs[0].ID)
// 	assert.Equal(t, "2", objs[1].ID)

// 	// Delete then List
// 	assert.Nil(t, cli.Delete(context.Background(), objs[0]))
// 	objs, err = cli.List(context.Background())
// 	assert.Nil(t, err)
// 	assert.Len(t, objs, 1)
// 	assert.Equal(t, "2", objs[0].ID)

// 	// Patch then List
// 	_, err = cli.Update(context.Background(), &v1.Ssl{
// 		ID:   "2",
// 		Snis: []string{"foo.com"},
// 	}, false)
// 	assert.Nil(t, err)
// 	objs, err = cli.List(context.Background())
// 	assert.Nil(t, err)
// 	assert.Len(t, objs, 1)
// 	assert.Equal(t, "2", objs[0].ID)
// 	assert.Equal(t, "foo.com", objs[0].Snis[0])
// }
