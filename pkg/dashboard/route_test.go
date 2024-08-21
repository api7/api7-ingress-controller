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
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

type fakeAPISIXRouteSrv struct {
	route map[string]map[string]interface{}
}

func (srv *fakeAPISIXRouteSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	uri := "/apisix/admin/routes/"

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/routes") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		HandleGet(uri, srv.route, w, r)
	case http.MethodDelete:
		HandleDelete(uri, srv.route, w, r)
	case http.MethodPut:
		HandlePut(uri, srv.route, w, r)
	case http.MethodPatch:
		HandlePut(uri, srv.route, w, r)
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
		baseURL:     u.String(),
		cli:         http.DefaultClient,
		cache:       &dummyCache{},
		cacheSynced: closedCh,
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
	})
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
	})
	assert.Nil(t, err)
	assert.Equal(t, "2", obj.ID)

	// List
	objs, err := cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.ElementsMatch(t, []string{"1", "2"}, []string{objs[0].ID, objs[1].ID})

	// Delete then List
	if objs[0].ID != "1" {
		objs[0], objs[1] = objs[1], objs[0]
	}
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
	})
	assert.Nil(t, err)
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].ID)
}
