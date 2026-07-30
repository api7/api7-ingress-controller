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

package translator

import (
	"github.com/go-logr/logr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

type Translator struct {
	Log                   logr.Logger
	ListenerPortMatchMode config.ListenerPortMatchMode
}

// normalizeMode resolves unset and unrecognized values to the default mode.
// server_port carries the port the data plane accepted the connection on
// (node_listen), not the port declared on the Gateway listener, so matching on
// it is only correct where the two coincide. That makes it opt-in.
func normalizeMode(mode config.ListenerPortMatchMode) config.ListenerPortMatchMode {
	switch mode {
	case config.ListenerPortMatchModeAuto, config.ListenerPortMatchModeExplicit, config.ListenerPortMatchModeOff:
		return mode
	default:
		return config.ListenerPortMatchModeOff
	}
}

func NewTranslator(log logr.Logger, mode config.ListenerPortMatchMode) *Translator {
	return &Translator{
		Log:                   log.WithName("translator"),
		ListenerPortMatchMode: normalizeMode(mode),
	}
}

// shouldInjectServerPortVars decides whether to pin StreamRoutes/routes to the
// matched listener port(s) via server_port.
//
// explicit reports whether the route attached to its Gateway through an explicit
// sectionName or port. It is computed by the controller from the matched
// RouteParentRefContext (provider.TranslateContext.HasExplicitListenerMatch),
// where each parentRef's Gateway and matched listeners are known, so an invalid
// explicit ref on one Gateway can never be satisfied by a same-named/ported
// listener matched through a different parentRef's Gateway.
func (t *Translator) shouldInjectServerPortVars(explicit bool, ports map[int32]struct{}) bool {
	if len(ports) == 0 {
		return false
	}

	switch t.ListenerPortMatchMode {
	case config.ListenerPortMatchModeExplicit:
		return explicit
	case config.ListenerPortMatchModeAuto:
		return explicit || len(ports) > 1
	default: // off, including anything normalizeMode resolved to it
		if explicit {
			t.Log.V(1).Info("listener_port_match_mode is 'off'; ignoring explicit listener targeting")
		}
		return false
	}
}

type TranslateResult struct {
	Services       []*adctypes.Service
	SSL            []*adctypes.SSL
	GlobalRules    adctypes.GlobalRule
	PluginMetadata adctypes.PluginMetadata
	Consumers      []*adctypes.Consumer
}
