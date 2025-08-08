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

package common

import (
	"maps"
	"sync"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

type ConfigManager[T any] struct {
	mu         sync.Mutex
	parentRefs map[types.NamespacedNameKind][]types.NamespacedNameKind
	configs    map[types.NamespacedNameKind]T
}

func NewConfigManager[T any]() *ConfigManager[T] {
	return &ConfigManager[T]{
		parentRefs: make(map[types.NamespacedNameKind][]types.NamespacedNameKind),
		configs:    make(map[types.NamespacedNameKind]T),
	}
}

func (s *ConfigManager[T]) GetParentRefs(rk types.NamespacedNameKind) []types.NamespacedNameKind {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.parentRefs[rk]
}

func (s *ConfigManager[T]) SetParentRefs(rk types.NamespacedNameKind, refs []types.NamespacedNameKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parentRefs[rk] = refs
}

func (s *ConfigManager[T]) Get(rk types.NamespacedNameKind) []T {
	s.mu.Lock()
	defer s.mu.Unlock()

	parentRefs := s.parentRefs[rk]
	configs := make([]T, 0, len(parentRefs))
	for _, parent := range parentRefs {
		if cfg, ok := s.configs[parent]; ok {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func (s *ConfigManager[T]) List() map[types.NamespacedNameKind]T {
	s.mu.Lock()
	defer s.mu.Unlock()

	configs := make(map[types.NamespacedNameKind]T, len(s.configs))
	maps.Copy(configs, s.configs)
	return configs
}

func (s *ConfigManager[T]) UpdateConfig(cfg T, parents ...types.NamespacedNameKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, parent := range parents {
		s.configs[parent] = cfg
	}
}

// buildConfig is a function that builds config of type T given a NamespacedNameKind.
func (s *ConfigManager[T]) Update(
	rk types.NamespacedNameKind,
	refs []types.NamespacedNameKind,
	buildConfig func(rk types.NamespacedNameKind) (*T, error),
) (discard []T, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldParentRefs := s.parentRefs[rk]
	s.parentRefs[rk] = refs

	parentRefSet := make(map[types.NamespacedNameKind]struct{})

	for _, ref := range refs {
		config, err := buildConfig(ref)
		if err != nil {
			return nil, err
		}
		if config == nil {
			zap.L().Debug("no config found for gateway proxy", zap.Any("parentRef", ref))
			continue
		}
		s.configs[ref] = *config
		parentRefSet[ref] = struct{}{}
	}

	for _, old := range oldParentRefs {
		if _, stillUsed := parentRefSet[old]; !stillUsed {
			if cfg, ok := s.configs[old]; ok {
				discard = append(discard, cfg)
			}
		}
	}

	return discard, nil
}

func (s *ConfigManager[T]) Set(parent types.NamespacedNameKind, cfg T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[parent] = cfg
}

func (s *ConfigManager[T]) Delete(rk types.NamespacedNameKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.parentRefs, rk)
	delete(s.configs, rk)
}

func (s *ConfigManager[T]) DeleteConfig(rks ...types.NamespacedNameKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rk := range rks {
		delete(s.configs, rk)
	}
}
