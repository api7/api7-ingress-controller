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

package cache

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-memdb"

	v1 "github.com/apache/apisix-ingress-controller/api/dashboard/v1"
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
					"label": {
						Name:         "label",
						Unique:       false,
						AllowMissing: true,
						Indexer: &LabelIndexer{
							LabelKeys: []string{"kind", "namespace", "name"},
							GetLabels: func(obj any) map[string]string {
								service, ok := obj.(*v1.Route)
								if !ok {
									return nil
								}
								return service.Labels
							},
						},
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
					"label": {
						Name:         "label",
						Unique:       false,
						AllowMissing: true,
						Indexer: &LabelIndexer{
							LabelKeys: []string{"kind", "namespace", "name"},
							GetLabels: func(obj any) map[string]string {
								service, ok := obj.(*v1.Service)
								if !ok {
									return nil
								}
								return service.Labels
							},
						},
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

// LabelIndexer is a custom indexer for exact match indexing
type LabelIndexer struct {
	LabelKeys []string
	GetLabels func(any) map[string]string
}

func (emi *LabelIndexer) FromObject(obj any) (bool, []byte, error) {
	labels := emi.GetLabels(obj)
	var labelValues []string
	for _, key := range emi.LabelKeys {
		if value, exists := labels[key]; exists {
			labelValues = append(labelValues, value)
		}
	}

	if len(labelValues) == 0 {
		return false, nil, nil
	}

	return true, []byte(strings.Join(labelValues, "/")), nil
}

func (emi *LabelIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != len(emi.LabelKeys) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(emi.LabelKeys), len(args))
	}

	labelValues := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("argument is not a string")
		}
		labelValues = append(labelValues, value)
	}

	return []byte(strings.Join(labelValues, "/")), nil
}
