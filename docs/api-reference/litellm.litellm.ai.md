# API Reference

## Packages
- [litellm.litellm.ai/v1alpha1](#litellmlitellmaiv1alpha1)


## litellm.litellm.ai/v1alpha1

Package v1alpha1 contains API Schema definitions for the litellm v1alpha1 API group.

### Resource Types
- [LiteLLMInstance](#litellminstance)
- [LiteLLMInstanceList](#litellminstancelist)



#### DatabaseSecretKeys







_Appears in:_
- [DatabaseSecretRef](#databasesecretref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hostSecret` _string_ |  |  |  |
| `passwordSecret` _string_ |  |  |  |
| `usernameSecret` _string_ |  |  |  |
| `dbnameSecret` _string_ |  |  |  |


#### DatabaseSecretRef







_Appears in:_
- [LiteLLMInstanceSpec](#litellminstancespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nameRef` _string_ |  |  |  |
| `keys` _[DatabaseSecretKeys](#databasesecretkeys)_ |  |  |  |


#### Gateway







_Appears in:_
- [LiteLLMInstanceSpec](#litellminstancespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |


#### Ingress







_Appears in:_
- [LiteLLMInstanceSpec](#litellminstancespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |


#### LiteLLMInstance



LiteLLMInstance is the Schema for the litellminstances API.



_Appears in:_
- [LiteLLMInstanceList](#litellminstancelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `litellm.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `LiteLLMInstance` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LiteLLMInstanceSpec](#litellminstancespec)_ |  |  |  |
| `status` _[LiteLLMInstanceStatus](#litellminstancestatus)_ |  |  |  |


#### LiteLLMInstanceList



LiteLLMInstanceList contains a list of LiteLLMInstance.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `litellm.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `LiteLLMInstanceList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[LiteLLMInstance](#litellminstance) array_ |  |  |  |


#### LiteLLMInstanceSpec



LiteLLMInstanceSpec defines the desired state of LiteLLMInstance.



_Appears in:_
- [LiteLLMInstance](#litellminstance)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ |  | ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1 |  |
| `masterKey` _string_ |  |  |  |
| `databaseSecretRef` _[DatabaseSecretRef](#databasesecretref)_ |  |  |  |
| `redisSecretRef` _[RedisSecretRef](#redissecretref)_ |  |  |  |
| `ingress` _[Ingress](#ingress)_ |  |  |  |
| `gateway` _[Gateway](#gateway)_ |  |  |  |


#### LiteLLMInstanceStatus



LiteLLMInstanceStatus defines the observed state of LiteLLMInstance.



_Appears in:_
- [LiteLLMInstance](#litellminstance)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration represents the .metadata.generation that the condition was set based upon |  |  |
| `lastUpdated` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#time-v1-meta)_ | LastUpdated represents the last time the status was updated |  |  |
| `configMapCreated` _boolean_ | Resource creation status |  |  |
| `secretCreated` _boolean_ |  |  |  |
| `deploymentCreated` _boolean_ |  |  |  |
| `serviceCreated` _boolean_ |  |  |  |
| `ingressCreated` _boolean_ |  |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#condition-v1-meta) array_ | Conditions represent the latest available observations of a LiteLLM instance's state |  |  |


#### RedisSecretKeys







_Appears in:_
- [RedisSecretRef](#redissecretref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hostSecret` _string_ |  |  |  |
| `portSecret` _integer_ |  |  |  |
| `passwordSecret` _string_ |  |  |  |


#### RedisSecretRef







_Appears in:_
- [LiteLLMInstanceSpec](#litellminstancespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nameRef` _string_ |  |  |  |
| `keys` _[RedisSecretKeys](#redissecretkeys)_ |  |  |  |


