#!/bin/bash

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Set default environment variables for APISIX standalone mode
export APISIX_IMAGE="${APISIX_IMAGE:-apache/apisix:3.8.0}"
export APISIX_ADMIN_KEY="${APISIX_ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
export APISIX_NAMESPACE="${APISIX_NAMESPACE:-apisix-standalone}"

echo "Starting APISIX standalone e2e tests..."
echo "APISIX Image: $APISIX_IMAGE"
echo "APISIX Admin Key: $APISIX_ADMIN_KEY"
echo "APISIX Namespace: $APISIX_NAMESPACE"

# Change to the project root directory
cd "$(dirname "$0")/.."

# Run the APISIX standalone e2e tests
go test -v -timeout=60m ./test/e2e -run TestAPISIXE2E

echo "APISIX standalone e2e tests completed." 
