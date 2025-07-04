<<<<<<< HEAD
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

=======
>>>>>>> release-v2-dev
# apisix-ingress-controller

## Description

<<<<<<< HEAD
The APISIX Ingress Controller allows you to run the APISIX Gateway as a
Kubernetes Ingress to handle inbound traffic for a Kubernetes cluster. It
dynamically configures and manages the APISIX Gateway using Gateway API
resources.
=======
The APISIX Ingress Controller allows you to run the APISIX Gateway as a Kubernetes Ingress to handle inbound traffic for a Kubernetes cluster. It dynamically configures and manages the API7 Gateway using Gateway API resources.
>>>>>>> release-v2-dev

## Document

* [Quickstart](./docs/quickstart.md)
* [Concepts](./docs/concepts.md)
* [Configuration](./docs/configuration.md)
* [Gateway API](./docs/gateway-api.md)

## Getting Started

### Prerequisites

* go version v1.22.0+
* docker version 17.03+.
* kubectl version v1.11.3+.
* Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make build-image
```

<<<<<<< HEAD
**NOTE:** This image ought to be published in the personal registry you
specified. And it is required to have access to pull the image from the
working environment. Make sure you have the proper permission to the registry
if the above commands don't work.
=======
**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.
>>>>>>> release-v2-dev

**Install the CRDs & Gateway API into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
<<<<<<< HEAD
make deploy #IMG=apache/apisix-ingress-controller:dev
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself
> cluster-admin privileges or be logged in as admin.
=======
make deploy #IMG=api7/api7-ingress-controller:dev
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.
>>>>>>> release-v2-dev

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

<<<<<<< HEAD
Following are the steps to build the installer and distribute this project to
users.
=======
Following are the steps to build the installer and distribute this project to users.
>>>>>>> release-v2-dev

1. Build the installer for the image built and published in the registry:

```sh
<<<<<<< HEAD
make build-installer # IMG=apache/apisix-ingress-controller:dev
=======
make build-installer # IMG=api7/api7-ingress-controller:dev
>>>>>>> release-v2-dev
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

<<<<<<< HEAD
1. Using the installer

Users can just run kubectl apply -f with the YAML bundle to install the
project, i.e.:
=======
2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:
>>>>>>> release-v2-dev

```sh
kubectl apply -f dist/install.yaml
```
<<<<<<< HEAD
=======

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
>>>>>>> release-v2-dev
