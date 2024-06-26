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
	"github.com/hashicorp/go-memdb"
)

var (
	_schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"route": {
				Name: "route",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"name": {
						Name:         "name",
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Name"},
						AllowMissing: true,
					},
					"service_id": {
						Name:         "service_id",
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "ServiceID"},
						AllowMissing: true,
					},
					"plugin_config_id": {
						Name:         "plugin_config_id",
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "PluginConfigId"},
						AllowMissing: true,
					},
				},
			},
			"service": {
				Name: "service",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"name": {
						Name:         "name",
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Name"},
						AllowMissing: true,
					},
				},
			},
			"ssl": {
				Name: "ssl",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
				},
			},
			"stream_route": {
				Name: "stream_route",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"service_id": {
						Name:         "service_id",
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "ServiceID"},
						AllowMissing: true,
					},
				},
			},
			"global_rule": {
				Name: "global_rule",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
				},
			},
			"consumer": {
				Name: "consumer",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Username"},
					},
				},
			},
			"schema": {
				Name: "schema",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"plugin_config": {
				Name: "plugin_config",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"name": {
						Name:         "name",
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Name"},
						AllowMissing: true,
					},
				},
			},
			"upstream_service": {
				Name: "upstream_service",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ServiceName"},
					},
				},
			},
		},
	}
)
