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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

func HandleGet(uri string, resource map[string]map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	var key string
	if strings.HasPrefix(r.URL.Path, "/apisix/admin/routes/") &&
		strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/") != "" {
		key = strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
	}
	if key != "" {
		resp := fakeGetCreateResp{
			fakeGetCreateItem{
				Key:   key,
				Value: resource[key],
			},
		}
		resp.fakeGetCreateItem.Value = resource[key]
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
	} else {
		resp := fakeListResp{}
		resp.Total = fmt.Sprintf("%d", len(resource))
		resp.List = make([]fakeListItem, 0, len(resource))
		for _, v := range resource {
			resp.List = append(resp.List, v)
		}
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
	}
}

func HandleDelete(uri string, resource map[string]map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, uri)
	id = uri + id
	code := http.StatusNotFound
	if _, ok := resource[id]; ok {
		delete(resource, id)
		code = http.StatusOK
	}
	w.WriteHeader(code)
}

func HandlePut(uri string, resource map[string]map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, uri)
	key := uri + id
	data, _ := io.ReadAll(r.Body)
	w.WriteHeader(http.StatusCreated)
	gr := make(map[string]interface{}, 0)
	_ = json.Unmarshal(data, &gr)
	resource[key] = gr
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
}

func HandlePatch(uri string, resource map[string]map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, uri)
	key := uri + id
	if _, ok := resource[key]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	data, _ := io.ReadAll(r.Body)
	var val Value
	_ = json.Unmarshal(data, &val)
	gr := make(map[string]interface{}, 0)
	_ = json.Unmarshal(data, &gr)
	resource[key] = gr
	w.WriteHeader(http.StatusOK)
	resp := fakeGetCreateResp{
		fakeGetCreateItem{
			Value: val,
			Key:   key,
		},
	}
	byt, _ := json.Marshal(resp)
	_, _ = w.Write(byt)
}

func TestAddCluster(t *testing.T) {
	apisix, err := NewClient()
	assert.Nil(t, err)

	err = apisix.AddCluster(context.Background(), &ClusterOptions{
		BaseURL: "http://service1:9080/apisix/admin",
	})
	assert.Nil(t, err)

	clusters := apisix.ListClusters()
	assert.Len(t, clusters, 1)

	err = apisix.AddCluster(context.Background(), &ClusterOptions{
		Name:    "service2",
		BaseURL: "http://service2:9080/apisix/admin",
	})
	assert.Nil(t, err)

	err = apisix.AddCluster(context.Background(), &ClusterOptions{
		Name:     "service2",
		AdminKey: "http://service3:9080/apisix/admin"})
	assert.Equal(t, ErrDuplicatedCluster, err)

	clusters = apisix.ListClusters()
	assert.Len(t, clusters, 2)
}

func TestNonExistentCluster(t *testing.T) {
	apisix, err := NewClient()
	assert.Nil(t, err)

	err = apisix.AddCluster(context.Background(), &ClusterOptions{
		BaseURL: "http://service1:9080/apisix/admin",
	})
	assert.Nil(t, err)

	_, err = apisix.Cluster("non-existent-cluster").Route().List(context.Background())
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").Route().Create(context.Background(), &v1.Route{})
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").Route().Update(context.Background(), &v1.Route{})
	assert.Equal(t, ErrClusterNotExist, err)
	err = apisix.Cluster("non-existent-cluster").Route().Delete(context.Background(), &v1.Route{})
	assert.Equal(t, ErrClusterNotExist, err)

	_, err = apisix.Cluster("non-existent-cluster").Service().List(context.Background())
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").Service().Create(context.Background(), &v1.Service{})
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").Service().Update(context.Background(), &v1.Service{})
	assert.Equal(t, ErrClusterNotExist, err)
	err = apisix.Cluster("non-existent-cluster").Service().Delete(context.Background(), &v1.Service{})
	assert.Equal(t, ErrClusterNotExist, err)

	_, err = apisix.Cluster("non-existent-cluster").SSL().List(context.Background())
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").SSL().Create(context.Background(), &v1.Ssl{})
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").SSL().Update(context.Background(), &v1.Ssl{})
	assert.Equal(t, ErrClusterNotExist, err)
	err = apisix.Cluster("non-existent-cluster").SSL().Delete(context.Background(), &v1.Ssl{})
	assert.Equal(t, ErrClusterNotExist, err)

	_, err = apisix.Cluster("non-existent-cluster").PluginConfig().List(context.Background())
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").PluginConfig().Create(context.Background(), &v1.PluginConfig{})
	assert.Equal(t, ErrClusterNotExist, err)
	_, err = apisix.Cluster("non-existent-cluster").PluginConfig().Update(context.Background(), &v1.PluginConfig{})
	assert.Equal(t, ErrClusterNotExist, err)
	err = apisix.Cluster("non-existent-cluster").PluginConfig().Delete(context.Background(), &v1.PluginConfig{})
	assert.Equal(t, ErrClusterNotExist, err)
}
