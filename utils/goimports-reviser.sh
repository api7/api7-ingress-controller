#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

set -e

go install github.com/incu6us/goimports-reviser/v2@latest

PROJECT_NAME=github.com/api7/api7-ingress-controller
while IFS= read -r -d '' file; do
  goimports-reviser  -file-path "$file" -project-name $PROJECT_NAME
done <   <(find . -name '*.go' -not -path "./test/*" -not -path "./pkg/kube/apisix/*" -print0)


PROJECT_NAME=github.com/api7/api7-ingress-controller/test/e2e
while IFS= read -r -d '' file; do
  goimports-reviser  -file-path "$file" -project-name $PROJECT_NAME
done <   <(find . -name '*.go' -path "./test/*" -print0)
