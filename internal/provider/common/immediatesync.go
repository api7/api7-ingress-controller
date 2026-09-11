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
	"fmt"
	"slices"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

// ResourceKeyLabels narrows labels down to just the resource-key label, which is what an
// immediate, single-resource push to ADC uses as its label selector: it must touch only
// the resources this one Kubernetes object contributed, not everything else that happens
// to share its kind, name, or namespace. The richer label set (kind, name, namespace)
// stays in the store's own bookkeeping, which keys entries by all of them.
func ResourceKeyLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	return map[string]string{label.LabelResourceKey: labels[label.LabelResourceKey]}
}

// WithMergedGlobalRules returns resources with GlobalRules replaced by every global_rule
// contribution currently in store for cfgName, merged into one -- but only when
// resourceTypes actually names "global_rule" as one of the types this push is scoped to.
//
// global_rule is a singleton object per config, not partitioned by label the way a route
// or a consumer is: an immediate push that carries only the just-translated object's own
// contribution would silently drop every other source's rules. The deferred, store-wide
// sync never needs this, since it already reads the whole store (see Store.GetResources).
func WithMergedGlobalRules(store *cache.Store, cfgName string, resourceTypes []string, resources *adctypes.Resources) (*adctypes.Resources, error) {
	if !slices.Contains(resourceTypes, adctypes.TypeGlobalRule) {
		return resources, nil
	}

	items, err := store.ListGlobalRules(cfgName)
	if err != nil {
		return nil, fmt.Errorf("failed to list global rules for config %s: %w", cfgName, err)
	}
	merged := make(adctypes.Plugins)
	for _, item := range items {
		for k, v := range item.Plugins {
			merged[k] = v
		}
	}

	out := &adctypes.Resources{}
	if resources != nil {
		*out = *resources
	}
	out.GlobalRules = adctypes.GlobalRule(merged)
	return out, nil
}

// PushError picks what to report for a failed immediate push: execErrs, when the data
// plane is what rejected it, carries the actual reason (e.g. "custom plugin
// (non-existent-plugin) not found") that a caller surfaces as a resource's status message.
// The generic err from syncConfigNow itself -- the build callback failing, say -- carries
// none of that, so it is only a fallback.
func PushError(cacheKey string, execErrs types.ADCExecutionErrors, err error) error {
	if len(execErrs.Errors) > 0 {
		return execErrs
	}
	return fmt.Errorf("config %s: %w", cacheKey, err)
}
