package cache

import (
	"github.com/hashicorp/go-memdb"
)

var (
	_schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
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
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
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
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
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
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
					},
				},
			},
		},
	}
)
