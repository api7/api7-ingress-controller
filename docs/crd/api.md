---
title: Resource Definitions API Reference
slug: /reference/api7-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the API7 Ingress Controller.
---

## Packages
- [apisix.apache.org/v1alpha1](#apisixapacheorgv1alpha1)


## apisix.apache.org/v1alpha1

Package v1alpha1 contains API Schema definitions for the apisix.apache.org v1alpha1 API group

- [BackendTrafficPolicy](#backendtrafficpolicy)
- [Consumer](#consumer)
- [GatewayProxy](#gatewayproxy)
- [HTTPRoutePolicy](#httproutepolicy)
- [PluginConfig](#pluginconfig)
### BackendTrafficPolicy




<!-- BackendTrafficPolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `BackendTrafficPolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[BackendTrafficPolicySpec](#backendtrafficpolicyspec)_ |  |



### Consumer




<!-- Consumer resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `Consumer`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ConsumerSpec](#consumerspec)_ |  |



### GatewayProxy


GatewayProxy is the Schema for the gatewayproxies API

<!-- GatewayProxy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `GatewayProxy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[GatewayProxySpec](#gatewayproxyspec)_ |  |



### HTTPRoutePolicy


HTTPRoutePolicy is the Schema for the httproutepolicies API.

<!-- HTTPRoutePolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `HTTPRoutePolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[HTTPRoutePolicySpec](#httproutepolicyspec)_ |  |



### PluginConfig


PluginConfig is the Schema for the PluginConfigs API

<!-- PluginConfig resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `PluginConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[PluginConfigSpec](#pluginconfigspec)_ |  |



### Types

In this section you will find types that the CRDs rely on.
#### AdminKeyAuth


AdminKeyAuth defines the admin key authentication configuration



| Field | Description |
| --- | --- |
| `value` _string_ | Value specifies the admin key value directly (not recommended for production) |
| `valueFrom` _[AdminKeyValueFrom](#adminkeyvaluefrom)_ | ValueFrom specifies the source of the admin key |


_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

#### AdminKeyValueFrom


AdminKeyValueFrom defines the source of the admin key



| Field | Description |
| --- | --- |
| `secretKeyRef` _[SecretKeySelector](#secretkeyselector)_ | SecretKeyRef references a key in a Secret |


_Appears in:_
- [AdminKeyAuth](#adminkeyauth)

#### AuthType
_Base type:_ `string`

AuthType defines the type of authentication





_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

#### BackendPolicyTargetReferenceWithSectionName
_Base type:_ `LocalPolicyTargetReferenceWithSectionName`





| Field | Description |
| --- | --- |
| `group` _[Group](#group)_ | Group is the group of the target resource. |
| `kind` _[Kind](#kind)_ | Kind is kind of the target resource. |
| `name` _[ObjectName](#objectname)_ | Name is the name of the target resource. |
| `sectionName` _[SectionName](#sectionname)_ | SectionName is the name of a section within the target resource. When unspecified, this targetRef targets the entire resource. In the following resources, SectionName is interpreted as the following:<br /><br /> * Gateway: Listener name * HTTPRoute: HTTPRouteRule name * Service: Port name<br /><br /> If a SectionName is specified, but does not exist on the targeted object, the Policy must fail to attach, and the policy implementation should record a `ResolvedRefs` or similar Condition in the Policy's status. |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### BackendTrafficPolicySpec






| Field | Description |
| --- | --- |
| `targetRefs` _[BackendPolicyTargetReferenceWithSectionName](#backendpolicytargetreferencewithsectionname) array_ | TargetRef identifies an API object to apply policy to. Currently, Backends (i.e. Service, ServiceImport, or any implementation-specific backendRef) are the only valid API target references. |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service. The default strategy is round robin. |
| `scheme` _string_ | The scheme used to talk with the upstream. |
| `retries` _integer_ | How many times that the proxy (Apache APISIX) should do when errors occur (error, timeout or bad http status codes like 500, 502). |
| `timeout` _[Timeout](#timeout)_ | Timeout settings for the read, send and connect to the upstream. |
| `passHost` _string_ | Configures the host when the request is forwarded to the upstream. Can be one of pass, node or rewrite. |
| `upstreamHost` _[Hostname](#hostname)_ | Specifies the host of the Upstream request. This is only valid if the passHost is set to rewrite |


_Appears in:_
- [BackendTrafficPolicy](#backendtrafficpolicy)

#### ConsumerSpec






| Field | Description |
| --- | --- |
| `gatewayRef` _[GatewayRef](#gatewayref)_ |  |
| `credentials` _[Credential](#credential) array_ |  |
| `plugins` _[Plugin](#plugin) array_ |  |


_Appears in:_
- [Consumer](#consumer)

#### ControlPlaneAuth


ControlPlaneAuth defines the authentication configuration for control plane



| Field | Description |
| --- | --- |
| `type` _[AuthType](#authtype)_ | Type specifies the type of authentication |
| `adminKey` _[AdminKeyAuth](#adminkeyauth)_ | AdminKey specifies the admin key authentication configuration |


_Appears in:_
- [ControlPlaneProvider](#controlplaneprovider)

#### ControlPlaneProvider


ControlPlaneProvider defines the configuration for control plane provider



| Field | Description |
| --- | --- |
| `endpoints` _string array_ | Endpoints specifies the list of control plane endpoints |
| `tlsVerify` _boolean_ | TlsVerify specifies whether to verify the TLS certificate of the control plane |
| `auth` _[ControlPlaneAuth](#controlplaneauth)_ | Auth specifies the authentication configuration |


_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

#### Credential






| Field | Description |
| --- | --- |
| `type` _string_ |  |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ |  |
| `secretRef` _[SecretReference](#secretreference)_ |  |
| `name` _string_ |  |


_Appears in:_
- [ConsumerSpec](#consumerspec)

#### GatewayProxyPlugin






| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `enabled` _boolean_ |  |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ |  |


_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

#### GatewayProxyProvider


GatewayProxyProvider defines the provider configuration for GatewayProxy



| Field | Description |
| --- | --- |
| `type` _[ProviderType](#providertype)_ | Type specifies the type of provider |
| `controlPlane` _[ControlPlaneProvider](#controlplaneprovider)_ | ControlPlane specifies the configuration for control plane provider |


_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

#### GatewayProxySpec


GatewayProxySpec defines the desired state of GatewayProxy



| Field | Description |
| --- | --- |
| `publishService` _string_ |  |
| `statusAddress` _string array_ |  |
| `provider` _[GatewayProxyProvider](#gatewayproxyprovider)_ |  |
| `plugins` _[GatewayProxyPlugin](#gatewayproxyplugin) array_ |  |
| `pluginMetadata` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io))_ |  |


_Appears in:_
- [GatewayProxy](#gatewayproxy)

#### GatewayRef






| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `kind` _string_ |  |
| `group` _string_ |  |
| `namespace` _string_ |  |


_Appears in:_
- [ConsumerSpec](#consumerspec)

#### HTTPRoutePolicySpec


HTTPRoutePolicySpec defines the desired state of HTTPRoutePolicy.



| Field | Description |
| --- | --- |
| `targetRefs` _LocalPolicyTargetReferenceWithSectionName array_ | TargetRef identifies an API object (enum: HTTPRoute, Ingress) to apply HTTPRoutePolicy to.<br /><br /> target references. |
| `priority` _integer_ |  |
| `vars` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io) array_ |  |


_Appears in:_
- [HTTPRoutePolicy](#httproutepolicy)

#### Hostname
_Base type:_ `string`







_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### LoadBalancer


LoadBalancer describes the load balancing parameters.



| Field | Description |
| --- | --- |
| `type` _string_ |  |
| `hashOn` _string_ | The HashOn and Key fields are required when Type is "chash". HashOn represents the key fetching scope. |
| `key` _string_ | Key represents the hash key. |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### Plugin






| Field | Description |
| --- | --- |
| `name` _string_ | The plugin name. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Plugin configuration. |


_Appears in:_
- [ConsumerSpec](#consumerspec)
- [PluginConfigSpec](#pluginconfigspec)

#### PluginConfigSpec


PluginConfigSpec defines the desired state of PluginConfig



| Field | Description |
| --- | --- |
| `plugins` _[Plugin](#plugin) array_ |  |


_Appears in:_
- [PluginConfig](#pluginconfig)



#### ProviderType
_Base type:_ `string`

ProviderType defines the type of provider





_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)



#### SecretReference






| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `namespace` _string_ |  |


_Appears in:_
- [Credential](#credential)



#### Timeout






| Field | Description |
| --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

