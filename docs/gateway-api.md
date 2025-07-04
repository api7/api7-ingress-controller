
# Gateway API

Gateway API is dedicated to achieving expressive and scalable Kubernetes service networking through various custom resources.

By supporting Gateway API, the APISIX Ingress controller can realize richer functions, including Gateway management, multi-cluster support, and other features. It is also possible to manage running instances of the APISIX gateway through Gateway API resource management.

## Concepts

- **GatewayClass**: Defines a set of Gateways that share a common configuration and behavior. Each GatewayClass is handled by a single controller, although controllers may handle more than one GatewayClass.
- **Gateway**: A resource in Kubernetes that describes how traffic can be translated to services within the cluster.
- **HTTPRoute**: Can be attached to a Gateway to configure HTTP

For more information about Gateway API, please refer to [Gateway API](https://gateway-api.sigs.k8s.io/).

## Gateway API Support Level

| Resource         | Core Support Level  | Extended Support Level | Implementation-Specific Support Level | API Version |
| ---------------- | ------------------- | ---------------------- | ------------------------------------- | ----------- |
| GatewayClass     | Supported           | N/A                    | Not supported                         | v1          |
| Gateway          | Partially supported | Partially supported    | Not supported                         | v1          |
| HTTPRoute        | Supported           | Partially supported    | Not supported                         | v1          |
| GRPCRoute        | Not supported       | Not supported          | Not supported                         | v1          |
| ReferenceGrant   | Not supported       | Not supported          | Not supported                         | v1beta1     |
| TLSRoute         | Not supported       | Not supported          | Not supported                         | v1alpha2    |
| TCPRoute         | Not supported       | Not supported          | Not supported                         | v1alpha2    |
| UDPRoute         | Not supported       | Not supported          | Not supported                         | v1alpha2    |
| BackendTLSPolicy | Not supported       | Not supported          | Not supported                         | v1alpha3    |

## HTTPRoute

The HTTPRoute resource allows users to configure HTTP routing by matching HTTP traffic and forwarding it to Kubernetes backends. Currently, the only backend supported by API7 Gateway is the Service resource.

### Example

The following example demonstrates how to configure an HTTPRoute resource to route traffic to the `httpbin` service:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"

---

apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
  namespace: default
spec:
  gatewayClassName: apisix
  listeners:
  - name: http
    protocol: HTTP
    port: 80

---

apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - backends.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin
      port: 80
```
