# Configure

The API7 Ingress Controller is a Kubernetes Ingress Controller that implements the Gateway API. This document describes how to configure the API7 Ingress Controller.

## Example

```yaml
log_level: "info"                               # The log level of the API7 Ingress Controller.
                                                # the default value is "info".

controller_name: gateway.apisix.io/api7-ingress-controller  # The controller name of the API7 Ingress Controller,
                                                          # which is used to identify the controller in the GatewayClass.
                                                          # The default value is "gateway.api7.io/api7-ingress-controller".

leader_election_id: "api7-ingress-controller-leader" # The leader election ID for the API7 Ingress Controller.
                                                        # The default value is "api7-ingress-controller-leader".
```

### Controller Name

The `controller_name` field is used to identify the `controllerName` in the GatewayClass.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: api7
spec:
  controllerName: "gateway.api7.io/api7-ingress-controller"
```

### Addresses

The `addresses` field records the status address of the Gateway.

```yaml
apiVersion: gateway.networking.k8s.io/v1
  kind: Gateway
  metadata:
    name: gateway1
  spec:
    gatewayClassName: api7
    listeners:
    - name: http
      port: 80
      protocol: HTTP
  status:
    addresses:
    - type: IPAddress
      value: 172.18.0.4
```
