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

package cache

import (
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

func TestMemDBCacheRoute(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	r1 := &v1.Route{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "abc",
		},
	}
	assert.Nil(t, c.InsertRoute(r1), "inserting route 1")

	r, err := c.GetRoute("1")
	assert.Nil(t, err)
	assert.Equal(t, r1, r)

	r2 := &v1.Route{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: "def",
		},
	}
	r3 := &v1.Route{
		Metadata: v1.Metadata{
			ID:   "3",
			Name: "ghi",
		},
	}
	assert.Nil(t, c.InsertRoute(r2), "inserting route r2")
	assert.Nil(t, c.InsertRoute(r3), "inserting route r3")

	r, err = c.GetRoute("3")
	assert.Nil(t, err)
	assert.Equal(t, r3, r)

	assert.Nil(t, c.DeleteRoute(r3), "delete route r3")

	routes, err := c.ListRoutes()
	assert.Nil(t, err, "listing routes")

	if routes[0].Name > routes[1].Name {
		routes[0], routes[1] = routes[1], routes[0]
	}
	assert.Equal(t, r1, routes[0])
	assert.Equal(t, r2, routes[1])

	r4 := &v1.Route{
		Metadata: v1.Metadata{
			ID:   "4",
			Name: "name4",
		},
	}
	assert.Error(t, ErrNotFound, c.DeleteRoute(r4))
}

func TestMemDBCacheSSL(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	s1 := &v1.Ssl{
		ID: "abc",
	}
	assert.Nil(t, c.InsertSSL(s1), "inserting ssl 1")

	s, err := c.GetSSL("abc")
	assert.Nil(t, err)
	assert.Equal(t, s1, s)

	s2 := &v1.Ssl{
		ID: "def",
	}
	s3 := &v1.Ssl{
		ID: "ghi",
	}
	assert.Nil(t, c.InsertSSL(s2), "inserting ssl 2")
	assert.Nil(t, c.InsertSSL(s3), "inserting ssl 3")

	s, err = c.GetSSL("ghi")
	assert.Nil(t, err)
	assert.Equal(t, s3, s)

	assert.Nil(t, c.DeleteSSL(s3), "delete ssl 3")

	ssl, err := c.ListSSL()
	assert.Nil(t, err, "listing ssl")

	if ssl[0].ID > ssl[1].ID {
		ssl[0], ssl[1] = ssl[1], ssl[0]
	}
	assert.Equal(t, s1, ssl[0])
	assert.Equal(t, s2, ssl[1])

	s4 := &v1.Ssl{
		ID: "id4",
	}
	assert.Error(t, ErrNotFound, c.DeleteSSL(s4))
}

func TestMemDBCacheUpstream(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	u1 := &v1.Service{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "abc",
		},
	}
	err = c.InsertService(u1)
	assert.Nil(t, err, "inserting upstream 1")

	u, err := c.GetService("1")
	assert.Nil(t, err)
	assert.Equal(t, u1, u)

	u2 := &v1.Service{
		Metadata: v1.Metadata{
			Name: "def",
			ID:   "2",
		},
	}
	u3 := &v1.Service{
		Metadata: v1.Metadata{
			Name: "ghi",
			ID:   "3",
		},
	}
	assert.Nil(t, c.InsertService(u2), "inserting upstream 2")
	assert.Nil(t, c.InsertService(u3), "inserting upstream 3")

	u, err = c.GetService("3")
	assert.Nil(t, err)
	assert.Equal(t, u3, u)

	assert.Nil(t, c.DeleteService(u3), "delete upstream 3")

	upstreams, err := c.ListServices()
	assert.Nil(t, err, "listing upstreams")

	if upstreams[0].Name > upstreams[1].Name {
		upstreams[0], upstreams[1] = upstreams[1], upstreams[0]
	}
	assert.Equal(t, u1, upstreams[0])
	assert.Equal(t, u2, upstreams[1])

	u4 := &v1.Service{
		Metadata: v1.Metadata{
			Name: "name4",
			ID:   "4",
		},
	}
	assert.Error(t, ErrNotFound, c.DeleteService(u4))
}

func TestMemDBCacheReference(t *testing.T) {
	r := &v1.Route{
		Metadata: v1.Metadata{
			Name: "route",
			ID:   "1",
		},
		ServiceID:      "1",
		PluginConfigId: "1",
	}
	u := &v1.Service{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "upstream",
		},
	}
	pc := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "pluginConfig",
		},
	}
	pc2 := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: "pluginConfig",
		},
	}

	db, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")
	assert.Nil(t, db.InsertRoute(r))
	assert.Nil(t, db.InsertService(u))
	assert.Nil(t, db.InsertPluginConfig(pc))

	assert.Error(t, ErrStillInUse, db.DeleteService(u))
	assert.Error(t, ErrStillInUse, db.DeletePluginConfig(pc))
	assert.Equal(t, memdb.ErrNotFound, db.DeletePluginConfig(pc2))
	assert.Nil(t, db.DeleteRoute(r))
	assert.Nil(t, db.DeleteService(u))
	assert.Nil(t, db.DeletePluginConfig(pc))
}

func testInsertAndGetGlobalRule(t *testing.T, c Cache, id string) {
	gr1 := &v1.GlobalRule{
		ID: id,
	}
	assert.Nil(t, c.InsertGlobalRule(gr1), "inserting global rule "+id)

	gr, err := c.GetGlobalRule(id)
	assert.Nil(t, err)
	assert.Equal(t, gr1, gr)
}

func TestMemDBCacheGlobalRule(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	testInsertAndGetGlobalRule(t, c, "1")
	testInsertAndGetGlobalRule(t, c, "2")
	testInsertAndGetGlobalRule(t, c, "3")

	grs, err := c.ListGlobalRules()
	assert.Nil(t, err, "listing global rules")
	assert.Len(t, grs, 3)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, []string{grs[0].ID, grs[1].ID, grs[2].ID})

	assert.Error(t, ErrNotFound, c.DeleteGlobalRule(&v1.GlobalRule{
		ID: "4",
	}))
}

func testInsertAndGetConsumer(t *testing.T, c Cache, username string) {
	c1 := &v1.Consumer{
		Username: username,
	}
	assert.Nil(t, c.InsertConsumer(c1), "inserting consumer "+username)

	c11, err := c.GetConsumer(username)
	assert.Nil(t, err)
	assert.Equal(t, c1, c11)
}

func TestMemDBCacheConsumer(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	testInsertAndGetConsumer(t, c, "jack")
	testInsertAndGetConsumer(t, c, "tom")
	testInsertAndGetConsumer(t, c, "jerry")
	consumers, err := c.ListConsumers()
	assert.Nil(t, err, "listing consumers")
	assert.Len(t, consumers, 3)

	assert.Nil(t, c.DeleteConsumer(
		&v1.Consumer{
			Username: "jerry",
		}), "delete consumer jerry")

	consumers, err = c.ListConsumers()
	assert.Nil(t, err, "listing consumers")
	assert.Len(t, consumers, 2)
	assert.ElementsMatch(t, []string{"jack", "tom"}, []string{consumers[0].Username, consumers[1].Username})

	assert.Error(t, ErrNotFound, c.DeleteConsumer(
		&v1.Consumer{
			Username: "chandler",
		},
	))
}

func TestMemDBCacheSchema(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	s1 := &v1.Schema{
		Name:    "plugins/p1",
		Content: "plugin schema",
	}
	assert.Nil(t, c.InsertSchema(s1), "inserting schema s1")

	s11, err := c.GetSchema("plugins/p1")
	assert.Nil(t, err)
	assert.Equal(t, s1, s11)

	s2 := &v1.Schema{
		Name: "plugins/p2",
	}
	s3 := &v1.Schema{
		Name: "plugins/p3",
	}
	assert.Nil(t, c.InsertSchema(s2), "inserting schema s2")
	assert.Nil(t, c.InsertSchema(s3), "inserting schema s3")

	s22, err := c.GetSchema("plugins/p2")
	assert.Nil(t, err)
	assert.Equal(t, s2, s22)

	assert.Nil(t, c.DeleteSchema(s3), "delete schema s3")

	schemaList, err := c.ListSchema()
	assert.Nil(t, err, "listing schema")

	if schemaList[0].Name > schemaList[1].Name {
		schemaList[0], schemaList[1] = schemaList[1], schemaList[0]
	}
	assert.Equal(t, s1, schemaList[0])
	assert.Equal(t, s2, schemaList[1])

	s4 := &v1.Schema{
		Name: "plugins/p4",
	}
	assert.Error(t, ErrNotFound, c.DeleteSchema(s4))
}

func TestMemDBCachePluginConfig(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	pc1 := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: "name1",
		},
	}
	assert.Nil(t, c.InsertPluginConfig(pc1), "inserting plugin_config pc1")

	pc11, err := c.GetPluginConfig("1")
	assert.Nil(t, err)
	assert.Equal(t, pc1, pc11)

	pc2 := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: "name2",
		},
	}
	pc3 := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "3",
			Name: "name3",
		},
	}
	assert.Nil(t, c.InsertPluginConfig(pc2), "inserting plugin_config pc2")
	assert.Nil(t, c.InsertPluginConfig(pc3), "inserting plugin_config pc3")

	pc22, err := c.GetPluginConfig("2")
	assert.Nil(t, err)
	assert.Equal(t, pc2, pc22)

	assert.Nil(t, c.DeletePluginConfig(pc3), "delete plugin_config pc3")

	pcList, err := c.ListPluginConfigs()
	assert.Nil(t, err, "listing plugin_config")

	if pcList[0].Name > pcList[1].Name {
		pcList[0], pcList[1] = pcList[1], pcList[0]
	}
	assert.Equal(t, pcList[0], pc1)
	assert.Equal(t, pcList[1], pc2)

	pc4 := &v1.PluginConfig{
		Metadata: v1.Metadata{
			ID:   "4",
			Name: "name4",
		},
	}
	assert.Error(t, ErrNotFound, c.DeletePluginConfig(pc4))
}
