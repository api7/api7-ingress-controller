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

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/api7/api7-ingress-controller/pkg/api/validation"
	"github.com/api7/api7-ingress-controller/pkg/apisix"
)

// MountWebhooks mounts webhook related routes.
func MountWebhooks(r *gin.Engine, co *apisix.ClusterOptions) {
	// init the schema client, it will be used to query schema of objects.
	_, _ = validation.GetSchemaClient(co)

	r.POST("/validate", validation.NewHandlerFunc("apisix", validation.Validator))

}
