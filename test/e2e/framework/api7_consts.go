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

package framework

import (
	"cmp"
	_ "embed"
	"os"
)

const (
	postgres     = "postgres"
	oceanbase    = "oceanbase"
	mysql        = "mysql"
	postgresDSN  = "postgres://api7ee:changeme@api7-postgresql:5432/api7ee"
	oceanbaseDSN = "mysql://root@tcp(oceanbase:2881)/api7ee"
	mysqlDSN     = "mysql://root:changeme@tcp(mysql:3306)/api7ee"
)

const (
	DashboardEndpoint    = "http://api7ee3-dashboard.api7-ee-e2e:7080"
	DashboardTLSEndpoint = "https://api7ee3-dashboard.api7-ee-e2e:7443"
	DPManagerTLSEndpoint = "https://api7ee3-dp-manager.api7-ee-e2e:7943"

	DashboardServiceName = "api7ee3-dashboard"
)

const (
	ProviderTypeAPI7EE = "api7ee"
)

const (
	Namespace = "api7-ee-e2e"
)

var (
	IngressVersion = cmp.Or(os.Getenv("INGRESS_VERSION"), "v1")
)
