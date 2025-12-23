# apisix-ingress-controller

## Description

The APISIX Ingress Controller allows you to run the APISIX Gateway as a Kubernetes Ingress to handle inbound traffic for a Kubernetes cluster. It dynamically configures and manages the API7 Gateway using Gateway API resources.

## Document

* [Getting Started](./docs/en/latest/getting-started)
* [Concepts](./docs/en/latest/concepts)
* [Configuration](./docs/en/latest/reference)

## Getting Started

### Prerequisites

<<<<<<< HEAD
* go version v1.23.0+
=======
* go version v1.22.0+.
>>>>>>> 01fa72f9 (docs: update k8s cluster version in README (#2688))
* docker version 17.03+.
* Kubernetes cluster version 1.26+.
* kubectl version within one minor version difference of your cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make build-image
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs & Gateway API into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy #IMG=api7/api7-ingress-controller:dev
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer # IMG=api7/api7-ingress-controller:dev
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f dist/install.yaml
```
