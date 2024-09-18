# Quickstart

This quickstart guide will help you get started with API7 Ingress Controller in a few simple steps.

## Prerequisites

* Kubernetes
* API7 Dashboard
* API7 Gateway

Please ensure you have deployed the API7 Dashboard control plane.

Note: Refer to the [Gateway API Release Changelog](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v1.0.0), it is recommended to use Kubernetes version 1.25+.

## Installation

Install the Gateway API CRDs:

```shell
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

```

Install The API7 Ingress Controller:

```shell
kubectl apply -f https://github.com/api7/api7-ingress-controller/releases/download/install.yaml

```

## Test HTTP Routing

Install the GatewayClass, Gateway, HTTPRoute and httpbin example app:

```shell
kubectl apply -f https://github.com/api7/api7-ingress-controller/blob/release-v2-dev/examples/quickstart.yaml
```

Requests will be forwarded by the gateway to the httpbin application:

```shell
curl http://{api7_gateway_loadbalancer_ip}/headers
```

:::Note If the API7 Gateway service without loadbalancer

You can forward the local port to the API7 Gateway service with the following command:

```shell
# Listen on port 9080 locally, forwarding to 80 in the pod
kubectl port-forward svc/${api7-gateway-svc} 9080:80 -n ${api7_gateway_namespace}
```

Now you can send HTTP requests to access it:

```shell
curl http://localhost:9080/headers
```

:::
