---
title: Configuration File
slug: /reference/apisix-ingress-controller/configuration-file
description: Configure the API7 Ingress Controller using config.yaml, including log settings, leader election, metrics, and synchronization behavior.
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

The API7 Ingress Controller uses a configuration file `config.yaml` to define core settings such as log level, leader election behavior, metrics endpoints, and synchronization intervals.

Configurations are defined in a Kubernetes ConfigMap and mounted into the controller pod as a file at runtime. To apply changes, you can update the ConfigMap and restart the controller Deployment to reload the configurations.

The example below shows binary defaults. Helm charts can render different values into the ConfigMap, so inspect the chart values and the generated ConfigMap when troubleshooting an installation.

```yaml
log_level: "info"                               # The log level of the API7 Ingress Controller.
                                                # The default value is "info".

controller_name: apisix.apache.org/apisix-ingress-controller  # The controller name of the API7 Ingress Controller,
                                                              # which is used to identify the controller in the GatewayClass.
                                                              # The default value is "apisix.apache.org/apisix-ingress-controller".
leader_election_id: "apisix-ingress-gateway-leader"       # The leader election ID for the API7 Ingress Controller.
                                                          # The default value is "apisix-ingress-gateway-leader".
leader_election:
  lease_duration: 30s                   # lease_duration is the duration that non-leader candidates will wait
                                        # after observing a leadership renewal until attempting to acquire leadership of a
                                        # leader election.
  renew_deadline: 20s                   # renew_deadline is the time in seconds that the acting controller
                                        # will retry refreshing leadership before giving up.
  retry_period: 2s                      # retry_period is the time in seconds that the acting controller
                                        # will wait between tries of actions with the controller.
  disable: false                        # Whether to disable leader election.

metrics_addr: ":8080"                   # The address the metrics endpoint binds to.
                                        # The default value is ":8080".

enable_server: false                    # Whether to enable the debug API.
                                        # The default value is false.
server_addr: ":9092"                    # The address the debug API binds to.
                                        # The default value is ":9092".

enable_http2: false                     # Whether to enable HTTP/2 for the server.
                                        # The default value is false.

probe_addr: ":8081"                     # The address the probe endpoint binds to.
                                        # The default value is ":8081".

secure_metrics: false                   # The secure metrics configuration.
                                        # The default value is "" (empty).

exec_adc_timeout: 15s                   # The timeout for the ADC to execute.
                                        # The default value is 15 seconds.

listener_port_match_mode: "off"         # Mode for injecting server_port route vars from Gateway listener ports.
                                        # - "off": never inject server_port vars.
                                        # - "auto": inject when parentRefs explicitly target listeners (sectionName/port) or when multiple listener ports are matched.
                                        # - "explicit": inject only when parentRefs explicitly target listeners.
                                        # The default value is "off".

provider:
  type: "api7ee"                        # Provider type.

  sync_period: 0s                       # The period between two consecutive syncs.
                                        # The default value is 0 seconds, which disables periodic synchronization.
                                        # Set it to a positive value to enable periodic synchronization.
  init_sync_delay: 20m                  # The initial delay before the first sync, only used when the controller is started.
                                        # The default value is 20 minutes.
```
