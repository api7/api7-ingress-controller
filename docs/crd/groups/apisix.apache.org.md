# API Reference

## Packages
- [apisix.apache.org/v1alpha1](#apisixapacheorgv1alpha1)


## apisix.apache.org/v1alpha1

Package v1alpha1 contains API Schema definitions for the apisix.apache.org v1alpha1 API group

### Resource Types
- [BackendTrafficPolicy](#backendtrafficpolicy)
- [Consumer](#consumer)
- [GatewayProxy](#gatewayproxy)
- [HTTPRoutePolicy](#httproutepolicy)
- [PluginConfig](#pluginconfig)



#### AdminKeyAuth



AdminKeyAuth defines the admin key authentication configuration



_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value specifies the admin key value directly (not recommended for production) |  |  |
| `valueFrom` _[AdminKeyValueFrom](#adminkeyvaluefrom)_ | ValueFrom specifies the source of the admin key |  |  |


#### AdminKeyValueFrom



AdminKeyValueFrom defines the source of the admin key



_Appears in:_
- [AdminKeyAuth](#adminkeyauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretKeyRef` _[SecretKeySelector](#secretkeyselector)_ | SecretKeyRef references a key in a Secret |  |  |


#### AuthType

_Underlying type:_ _string_

AuthType defines the type of authentication

_Validation:_
- Enum: [AdminKey]

_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

| Field | Description |
| --- | --- |
| `AdminKey` | AuthTypeAdminKey represents the admin key authentication type<br /> |


#### BackendPolicyTargetReferenceWithSectionName

_Underlying type:_ _LocalPolicyTargetReferenceWithSectionName_





_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `group` _[Group](#group)_ | Group is the group of the target resource. |  | MaxLength: 253 <br />Pattern: `^$\|^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `kind` _[Kind](#kind)_ | Kind is kind of the target resource. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-zA-Z]([-a-zA-Z0-9]*[a-zA-Z0-9])?$` <br /> |
| `name` _[ObjectName](#objectname)_ | Name is the name of the target resource. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `sectionName` _[SectionName](#sectionname)_ | SectionName is the name of a section within the target resource. When<br />unspecified, this targetRef targets the entire resource. In the following<br />resources, SectionName is interpreted as the following:<br /><br />* Gateway: Listener name<br />* HTTPRoute: HTTPRouteRule name<br />* Service: Port name<br /><br />If a SectionName is specified, but does not exist on the targeted object,<br />the Policy must fail to attach, and the policy implementation should record<br />a `ResolvedRefs` or similar Condition in the Policy's status. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |


#### BackendTrafficPolicy









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1` | | |
| `kind` _string_ | `BackendTrafficPolicy` | | |
| `spec` _[BackendTrafficPolicySpec](#backendtrafficpolicyspec)_ |  |  |  |


#### BackendTrafficPolicySpec







_Appears in:_
- [BackendTrafficPolicy](#backendtrafficpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `targetRefs` _[BackendPolicyTargetReferenceWithSectionName](#backendpolicytargetreferencewithsectionname) array_ | TargetRef identifies an API object to apply policy to.<br />Currently, Backends (i.e. Service, ServiceImport, or any<br />implementation-specific backendRef) are the only valid API<br />target references. |  | MaxItems: 16 <br />MinItems: 1 <br /> |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service.<br />The default strategy is round robin. |  |  |
| `scheme` _string_ | The scheme used to talk with the upstream. | http | Enum: [http https grpc grpcs] <br /> |
| `retries` _integer_ | How many times that the proxy (Apache APISIX) should do when<br />errors occur (error, timeout or bad http status codes like 500, 502). |  |  |
| `timeout` _[Timeout](#timeout)_ | Timeout settings for the read, send and connect to the upstream. |  |  |
| `passHost` _string_ | Configures the host when the request is forwarded to the upstream.<br />Can be one of pass, node or rewrite. | pass | Enum: [pass node rewrite] <br /> |
| `upstreamHost` _[Hostname](#hostname)_ | Specifies the host of the Upstream request. This is only valid if<br />the passHost is set to rewrite |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^(\*\.)?[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |


#### Consumer









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1` | | |
| `kind` _string_ | `Consumer` | | |
| `spec` _[ConsumerSpec](#consumerspec)_ |  |  |  |


#### ConsumerSpec







_Appears in:_
- [Consumer](#consumer)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `gatewayRef` _[GatewayRef](#gatewayref)_ |  |  |  |
| `credentials` _[Credential](#credential) array_ |  |  |  |
| `plugins` _[Plugin](#plugin) array_ |  |  |  |


#### ControlPlaneAuth



ControlPlaneAuth defines the authentication configuration for control plane



_Appears in:_
- [ControlPlaneProvider](#controlplaneprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[AuthType](#authtype)_ | Type specifies the type of authentication |  | Enum: [AdminKey] <br />Required: \{\} <br /> |
| `adminKey` _[AdminKeyAuth](#adminkeyauth)_ | AdminKey specifies the admin key authentication configuration |  |  |


#### ControlPlaneProvider



ControlPlaneProvider defines the configuration for control plane provider



_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoints` _string array_ | Endpoints specifies the list of control plane endpoints |  | MinItems: 1 <br />Required: \{\} <br /> |
| `tlsVerify` _boolean_ | TlsVerify specifies whether to verify the TLS certificate of the control plane |  |  |
| `auth` _[ControlPlaneAuth](#controlplaneauth)_ | Auth specifies the authentication configuration |  | Required: \{\} <br /> |


#### Credential







_Appears in:_
- [ConsumerSpec](#consumerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [jwt-auth basic-auth key-auth hmac-auth] <br />Required: \{\} <br /> |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#json-v1-apiextensions-k8s-io)_ |  |  |  |
| `secretRef` _[SecretReference](#secretreference)_ |  |  |  |
| `name` _string_ |  |  |  |


#### GatewayProxy



GatewayProxy is the Schema for the gatewayproxies API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1` | | |
| `kind` _string_ | `GatewayProxy` | | |
| `spec` _[GatewayProxySpec](#gatewayproxyspec)_ |  |  |  |


#### GatewayProxyPlugin







_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#json-v1-apiextensions-k8s-io)_ |  |  |  |


#### GatewayProxyProvider



GatewayProxyProvider defines the provider configuration for GatewayProxy



_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ProviderType](#providertype)_ | Type specifies the type of provider |  | Enum: [ControlPlane] <br />Required: \{\} <br /> |
| `controlPlane` _[ControlPlaneProvider](#controlplaneprovider)_ | ControlPlane specifies the configuration for control plane provider |  |  |


#### GatewayProxySpec



GatewayProxySpec defines the desired state of GatewayProxy



_Appears in:_
- [GatewayProxy](#gatewayproxy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `publishService` _string_ |  |  |  |
| `statusAddress` _string array_ |  |  |  |
| `provider` _[GatewayProxyProvider](#gatewayproxyprovider)_ |  |  |  |
| `plugins` _[GatewayProxyPlugin](#gatewayproxyplugin) array_ |  |  |  |
| `pluginMetadata` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#json-v1-apiextensions-k8s-io))_ |  |  |  |


#### GatewayRef







_Appears in:_
- [ConsumerSpec](#consumerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br />Required: \{\} <br /> |
| `kind` _string_ |  | Gateway |  |
| `group` _string_ |  | gateway.networking.k8s.io |  |
| `namespace` _string_ |  |  |  |


#### HTTPRoutePolicy



HTTPRoutePolicy is the Schema for the httproutepolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1` | | |
| `kind` _string_ | `HTTPRoutePolicy` | | |
| `spec` _[HTTPRoutePolicySpec](#httproutepolicyspec)_ |  |  |  |


#### HTTPRoutePolicySpec



HTTPRoutePolicySpec defines the desired state of HTTPRoutePolicy.



_Appears in:_
- [HTTPRoutePolicy](#httproutepolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `targetRefs` _LocalPolicyTargetReferenceWithSectionName array_ | TargetRef identifies an API object (enum: HTTPRoute, Ingress) to apply HTTPRoutePolicy to.<br /><br />target references. |  | MaxItems: 16 <br />MinItems: 1 <br /> |
| `priority` _integer_ |  |  |  |
| `vars` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#json-v1-apiextensions-k8s-io) array_ |  |  |  |


#### Hostname

_Underlying type:_ _string_



_Validation:_
- MaxLength: 253
- MinLength: 1
- Pattern: `^(\*\.)?[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`

_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)



#### LoadBalancer



LoadBalancer describes the load balancing parameters.



_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  | roundrobin | Enum: [roundrobin chash ewma least_conn] <br />Required: \{\} <br /> |
| `hashOn` _string_ | The HashOn and Key fields are required when Type is "chash".<br />HashOn represents the key fetching scope. | vars | Enum: [vars header cookie consumer vars_combinations] <br /> |
| `key` _string_ | Key represents the hash key. |  |  |


#### Plugin







_Appears in:_
- [ConsumerSpec](#consumerspec)
- [PluginConfigSpec](#pluginconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The plugin name. |  |  |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#json-v1-apiextensions-k8s-io)_ | Plugin configuration. |  |  |


#### PluginConfig



PluginConfig is the Schema for the PluginConfigs API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1` | | |
| `kind` _string_ | `PluginConfig` | | |
| `spec` _[PluginConfigSpec](#pluginconfigspec)_ |  |  |  |


#### PluginConfigSpec



PluginConfigSpec defines the desired state of PluginConfig



_Appears in:_
- [PluginConfig](#pluginconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `plugins` _[Plugin](#plugin) array_ |  |  |  |


#### ProviderType

_Underlying type:_ _string_

ProviderType defines the type of provider

_Validation:_
- Enum: [ControlPlane]

_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

| Field | Description |
| --- | --- |
| `ControlPlane` | ProviderTypeControlPlane represents the control plane provider type<br /> |




#### SecretReference







_Appears in:_
- [Credential](#credential)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `namespace` _string_ |  |  |  |


#### Timeout







_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#duration-v1-meta)_ |  | 60s | Pattern: `^[0-9]+s$` <br />Type: string <br /> |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#duration-v1-meta)_ |  | 60s | Pattern: `^[0-9]+s$` <br />Type: string <br /> |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#duration-v1-meta)_ |  | 60s | Pattern: `^[0-9]+s$` <br />Type: string <br /> |


