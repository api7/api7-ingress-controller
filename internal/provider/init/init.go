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

package init

import (
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/api7ee"
	"github.com/apache/apisix-ingress-controller/internal/provider/apisix"
	"github.com/go-logr/logr"
)

func init() {
	provider.Register("apisix", apisix.New)
<<<<<<< HEAD
	provider.Register("apisix-standalone", func(statusUpdater status.Updater, readinessManager readiness.ReadinessManager, opts ...provider.Option) (provider.Provider, error) {
		opts = append(opts, provider.WithBackendMode("apisix-standalone"))
		opts = append(opts, provider.WithResolveEndpoints())
		return apisix.New(statusUpdater, readinessManager, opts...)
	})
	provider.Register("api7ee", api7ee.New)
=======
	provider.Register("apisix-standalone",
		func(log logr.Logger,
			statusUpdater status.Updater,
			readinessManager readiness.ReadinessManager,
			opts ...provider.Option,
		) (provider.Provider, error) {
			opts = append(opts, provider.WithBackendMode("apisix-standalone"))
			opts = append(opts, provider.WithResolveEndpoints())
			return apisix.New(log, statusUpdater, readinessManager, opts...)
		})
>>>>>>> d9550d88 (chore: unify the logging component (#2584))
}
