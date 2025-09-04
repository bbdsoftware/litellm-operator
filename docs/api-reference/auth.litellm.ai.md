# API Reference

## Packages
- [auth.litellm.ai/v1alpha1](#authlitellmaiv1alpha1)


## auth.litellm.ai/v1alpha1

Package v1alpha1 contains API Schema definitions for the auth v1alpha1 API group

### Resource Types
- [Team](#team)
- [TeamList](#teamlist)
- [TeamMemberAssociation](#teammemberassociation)
- [TeamMemberAssociationList](#teammemberassociationlist)
- [User](#user)
- [UserList](#userlist)
- [VirtualKey](#virtualkey)
- [VirtualKeyList](#virtualkeylist)



#### ConnectionRef



ConnectionRef defines how to connect to a LiteLLM instance



_Appears in:_
- [TeamMemberAssociationSpec](#teammemberassociationspec)
- [TeamSpec](#teamspec)
- [UserSpec](#userspec)
- [VirtualKeySpec](#virtualkeyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretRef](#secretref)_ | SecretRef references a secret containing connection details |  |  |
| `instanceRef` _[InstanceRef](#instanceref)_ | InstanceRef references a LiteLLM instance |  |  |


#### InstanceRef



InstanceRef references a LiteLLM instance



_Appears in:_
- [ConnectionRef](#connectionref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LiteLLM instance |  |  |
| `namespace` _string_ | Namespace is the namespace of the LiteLLM instance (defaults to the same namespace as the Team) |  |  |


#### SecretKeys



SecretKeys defines the keys in a secret for connection details



_Appears in:_
- [SecretRef](#secretref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `masterKey` _string_ | MasterKey is the key in the secret containing the master key |  |  |
| `url` _string_ | URL is the key in the secret containing the LiteLLM URL |  |  |


#### SecretRef



SecretRef references a secret containing connection details



_Appears in:_
- [ConnectionRef](#connectionref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the secret |  |  |
| `keys` _[SecretKeys](#secretkeys)_ | Keys defines the keys in the secret that contain connection details |  |  |


#### Team



Team is the Schema for the teams API



_Appears in:_
- [TeamList](#teamlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `Team` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[TeamSpec](#teamspec)_ |  |  |  |
| `status` _[TeamStatus](#teamstatus)_ |  |  |  |


#### TeamList



TeamList contains a list of Team





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `TeamList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Team](#team) array_ |  |  |  |


#### TeamMemberAssociation



TeamMemberAssociation is the Schema for the teammemberassociations API



_Appears in:_
- [TeamMemberAssociationList](#teammemberassociationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `TeamMemberAssociation` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[TeamMemberAssociationSpec](#teammemberassociationspec)_ |  |  |  |
| `status` _[TeamMemberAssociationStatus](#teammemberassociationstatus)_ |  |  |  |


#### TeamMemberAssociationList



TeamMemberAssociationList contains a list of TeamMemberAssociation





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `TeamMemberAssociationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[TeamMemberAssociation](#teammemberassociation) array_ |  |  |  |


#### TeamMemberAssociationSpec



TeamMemberAssociationSpec defines the desired state of TeamMemberAssociation



_Appears in:_
- [TeamMemberAssociation](#teammemberassociation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[ConnectionRef](#connectionref)_ | ConnectionRef defines how to connect to the LiteLLM instance |  | Required: \{\} <br /> |
| `maxBudgetInTeam` _string_ | MaxBudgetInTeam is the maximum budget for the user in the team |  |  |
| `teamAlias` _string_ | TeamID is the ID of the team |  | Required: \{\} <br /> |
| `userEmail` _string_ | UserEmail is the email of the user |  | Required: \{\} <br /> |
| `role` _string_ | Role is the role of the user - one of "admin" or "user" |  | Enum: [admin user] <br />Required: \{\} <br /> |


#### TeamMemberAssociationStatus



TeamMemberAssociationStatus defines the observed state of TeamMemberAssociation



_Appears in:_
- [TeamMemberAssociation](#teammemberassociation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `teamAlias` _string_ | TeamAlias is the alias of the team |  |  |
| `teamID` _string_ | TeamID is the ID of the team |  |  |
| `userEmail` _string_ | UserEmail is the email of the user |  |  |
| `userID` _string_ | UserID is the ID of the user |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#condition-v1-meta) array_ |  |  |  |


#### TeamMemberWithRole







_Appears in:_
- [TeamStatus](#teamstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `userID` _string_ | UserID is the ID of the user |  |  |
| `userEmail` _string_ | UserEmail is the email of the user |  |  |
| `role` _string_ | Role is the role of the user - one of "admin" or "user" |  |  |


#### TeamSpec



TeamSpec defines the desired state of Team



_Appears in:_
- [Team](#team)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[ConnectionRef](#connectionref)_ | ConnectionRef defines how to connect to the LiteLLM instance |  | Required: \{\} <br /> |
| `blocked` _boolean_ | Blocked is a flag indicating if the team is blocked or not - will stop all calls from keys with this team_id |  |  |
| `budgetDuration` _string_ | BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo"). |  |  |
| `guardrails` _string array_ | Guardrails are guardrails for the team |  |  |
| `maxBudget` _string_ | MaxBudget is the maximum budget for the team |  |  |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `modelAliases` _object (keys:string, values:string)_ | ModelAliases are model aliases for the team |  |  |
| `models` _string array_ | Models is the list of models that are associated with the team. All keys for this team_id will have at most, these models. If empty, assumes all models are allowed. |  |  |
| `organizationID` _string_ | OrganizationID is the ID of the organization that the team belongs to. If not set, the team will be created with no organization. |  |  |
| `rpmLimit` _integer_ | RPMLimit is the maximum requests per minute limit for the team - all keys associated with this team_id will have at max this RPM limit |  |  |
| `tags` _string array_ | Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license |  |  |
| `teamAlias` _string_ | TeamAlias is the alias of the team |  | Required: \{\} <br /> |
| `teamID` _string_ | TeamID is the ID of the team. If not set, a unique ID will be generated. |  |  |
| `teamMemberPermissions` _string array_ | TeamMemberPermissions is the list of routes that non-admin team members can access. Example: ["/key/generate", "/key/update", "/key/delete"] |  |  |
| `tpmLimit` _integer_ | TPMLimit is the maximum tokens per minute limit for the team - all keys with this team_id will have at max this TPM limit |  |  |


#### TeamStatus



TeamStatus defines the observed state of Team



_Appears in:_
- [Team](#team)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `blocked` _boolean_ | Blocked is a flag indicating if the team is blocked or not |  |  |
| `budgetDuration` _string_ | BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. |  |  |
| `budgetResetAt` _string_ | BudgetResetAt is the date and time when the budget will be reset |  |  |
| `createdAt` _string_ | CreatedAt is the date and time when the team was created |  |  |
| `liteLLMModelTable` _string_ | LiteLLMModelTable is the model table for the team |  |  |
| `maxBudget` _string_ | MaxBudget is the maximum budget for the team |  |  |
| `maxParallelRequests` _integer_ | MaxParallelRequests is the maximum number of parallel requests allowed |  |  |
| `membersWithRole` _[TeamMemberWithRole](#teammemberwithrole) array_ | MembersWithRole is the list of members with role |  |  |
| `modelID` _string_ | ModelID is the ID of the model |  |  |
| `models` _string array_ | Models is the list of models that are associated with the team. All keys for this team_id will have at most, these models. |  |  |
| `organizationID` _string_ | OrganizationID is the ID of the organization that the team belongs to |  |  |
| `rpmLimit` _integer_ | RPMLimit is the maximum requests per minute limit for the team - all keys associated with this team_id will have at max this RPM limit |  |  |
| `spend` _string_ | Spend is the current spend of the team |  |  |
| `tags` _string array_ | Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license |  |  |
| `teamAlias` _string_ | TeamAlias is the alias of the team |  |  |
| `teamID` _string_ | TeamID is the ID of the team |  |  |
| `teamMemberPermissions` _string array_ | TeamMemberPermissions is the list of routes that non-admin team members can access |  |  |
| `tpmLimit` _integer_ | TPMLimit is the maximum tokens per minute limit for the team - all keys with this team_id will have at max this TPM limit |  |  |
| `updatedAt` _string_ | UpdatedAt is the date and time when the team was last updated |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#condition-v1-meta) array_ |  |  |  |


#### User



User is the Schema for the users API



_Appears in:_
- [UserList](#userlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `User` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UserSpec](#userspec)_ |  |  |  |
| `status` _[UserStatus](#userstatus)_ |  |  |  |


#### UserList



UserList contains a list of User





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `UserList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[User](#user) array_ |  |  |  |


#### UserSpec



UserSpec defines the desired state of User



_Appears in:_
- [User](#user)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[ConnectionRef](#connectionref)_ | ConnectionRef defines how to connect to the LiteLLM instance |  | Required: \{\} <br /> |
| `aliases` _object (keys:string, values:string)_ | Aliases is the model aliases for the user |  |  |
| `allowedCacheControls` _string array_ | AllowedCacheControls is the list of allowed cache control values |  |  |
| `autoCreateKey` _boolean_ | AutoCreateKey is whether to automatically create a key for the user |  |  |
| `blocked` _boolean_ | Blocked is whether the user is blocked |  |  |
| `budgetDuration` _string_ | BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo"). |  |  |
| `duration` _string_ | Duration is the duration for the key auto-created on /user/new |  |  |
| `guardrails` _string array_ | Guardrails is the list of active guardrails for the user |  |  |
| `keyAlias` _string_ | KeyAlias is the optional alias of the key if autoCreateKey is true |  |  |
| `maxBudget` _string_ | MaxBudget is the maximum budget for the user |  |  |
| `maxParallelRequests` _integer_ | MaxParallelRequests is the maximum number of parallel requests for the user |  |  |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `modelMaxBudget` _object (keys:string, values:string)_ | ModelMaxBudget is the model specific maximum budget |  |  |
| `modelRPMLimit` _object (keys:string, values:string)_ | ModelRPMLimit is the model specific maximum requests per minute |  |  |
| `modelTPMLimit` _object (keys:string, values:string)_ | ModelTPMLimit is the model specific maximum tokens per minute |  |  |
| `models` _string array_ | Models is the list of models that the user is allowed to use |  |  |
| `permissions` _object (keys:string, values:string)_ | Permissions is the user-specific permissions |  |  |
| `rpmLimit` _integer_ | RPMLimit is the maximum requests per minute for the user |  |  |
| `sendInviteEmail` _boolean_ | SendInviteEmail is whether to send an invite email to the user - NOTE: the user endpoint will return an error if email alerting is not configured and this is enabled, but the user will still be created. |  |  |
| `softBudget` _string_ | SoftBudget - alert when user exceeds this budget, doesn't block requests |  |  |
| `spend` _string_ | Spend is the amount spent by user |  |  |
| `ssoUserID` _string_ | SSOUserID is the id of the user in the SSO provider |  |  |
| `teams` _string array_ | Teams is the list of teams that the user is a member of |  |  |
| `tpmLimit` _integer_ | TPMLimit is the maximum tokens per minute for the user |  |  |
| `userAlias` _string_ | UserAlias is the alias of the user |  |  |
| `userEmail` _string_ | UserEmail is the email of the user |  | Required: \{\} <br /> |
| `userID` _string_ | UserID is the ID of the user. If not set, a unique ID will be generated. |  |  |
| `userRole` _string_ | UserRole is the role of the user - one of "proxy_admin", "proxy_admin_viewer", "internal_user", "internal_user_viewer" |  |  |


#### UserStatus



UserStatus defines the observed state of User



_Appears in:_
- [User](#user)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedCacheControls` _string array_ | AllowedCacheControls is the list of allowed cache control values |  |  |
| `allowedRoutes` _string array_ | AllowedRoutes is the list of allowed routes |  |  |
| `aliases` _object (keys:string, values:string)_ | Aliases is the model aliases for the user |  |  |
| `blocked` _boolean_ | Blocked is whether the user is blocked |  |  |
| `budgetDuration` _string_ | BudgetDuration - Budget is reset at the end of specified duration |  |  |
| `budgetID` _string_ | BudgetID is the ID of the budget |  |  |
| `config` _object (keys:string, values:string)_ | Config is the user-specific config |  |  |
| `createdAt` _string_ | CreatedAt is the date and time when the user was created |  |  |
| `createdBy` _string_ | CreatedBy is the user who created this user |  |  |
| `duration` _string_ | Duration is the duration for the key |  |  |
| `enforcedParams` _string array_ | EnforcedParams is the list of enforced parameters |  |  |
| `expires` _string_ | Expires is the date and time when the user will expire |  |  |
| `guardrails` _string array_ | Guardrails is the list of active guardrails |  |  |
| `keyAlias` _string_ | KeyAlias is the alias of the key |  |  |
| `keyName` _string_ | KeyName is the name of the key |  |  |
| `keySecretRef` _string_ | KeySecretRef is the reference to the secret containing the user key |  |  |
| `litellmBudgetTable` _string_ | LiteLLMBudgetTable is the budget table name |  |  |
| `maxBudget` _string_ | MaxBudget is the maximum budget for the user |  |  |
| `maxParallelRequests` _integer_ | MaxParallelRequests is the maximum number of parallel requests |  |  |
| `modelMaxBudget` _object (keys:string, values:string)_ | ModelMaxBudget is the model specific maximum budget |  |  |
| `modelRPMLimit` _object (keys:string, values:string)_ | ModelRPMLimit is the model specific maximum requests per minute |  |  |
| `modelTPMLimit` _object (keys:string, values:string)_ | ModelTPMLimit is the model specific maximum tokens per minute |  |  |
| `models` _string array_ | Models is the list of models that the user is allowed to use |  |  |
| `permissions` _object (keys:string, values:string)_ | Permissions is the user-specific permissions |  |  |
| `rpmLimit` _integer_ | RPMLimit is the maximum requests per minute |  |  |
| `spend` _string_ | Spend is the amount spent by user |  |  |
| `tags` _string array_ | Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license |  |  |
| `teams` _string array_ | Teams is the list of teams that the user is a member of |  |  |
| `token` _string_ | Token is the user's token |  |  |
| `tpmLimit` _integer_ | TPMLimit is the maximum tokens per minute |  |  |
| `updatedAt` _string_ | UpdatedAt is the date and time when the user was last updated |  |  |
| `updatedBy` _string_ | UpdatedBy is the user who last updated this user |  |  |
| `userAlias` _string_ | UserAlias is the alias of the user |  |  |
| `userEmail` _string_ | UserEmail is the email of the user |  |  |
| `userID` _string_ | UserID is the unique user id |  |  |
| `userRole` _string_ | UserRole is the role of the user |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#condition-v1-meta) array_ |  |  |  |


#### VirtualKey



VirtualKey is the Schema for the virtualkeys API



_Appears in:_
- [VirtualKeyList](#virtualkeylist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `VirtualKey` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VirtualKeySpec](#virtualkeyspec)_ |  |  |  |
| `status` _[VirtualKeyStatus](#virtualkeystatus)_ |  |  |  |


#### VirtualKeyList



VirtualKeyList contains a list of VirtualKey





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `auth.litellm.ai/v1alpha1` | | |
| `kind` _string_ | `VirtualKeyList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VirtualKey](#virtualkey) array_ |  |  |  |


#### VirtualKeySpec



VirtualKeySpec defines the desired state of VirtualKey



_Appears in:_
- [VirtualKey](#virtualkey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[ConnectionRef](#connectionref)_ | ConnectionRef defines how to connect to the LiteLLM instance |  | Required: \{\} <br /> |
| `aliases` _object (keys:string, values:string)_ | Aliases maps additional aliases for the key |  |  |
| `allowedCacheControls` _string array_ | AllowedCacheControls defines allowed cache control settings |  |  |
| `allowedRoutes` _string array_ | AllowedRoutes defines allowed API routes |  |  |
| `blocked` _boolean_ | Blocked indicates if the key is blocked |  |  |
| `budgetDuration` _string_ | BudgetDuration specifies the duration for budget tracking |  |  |
| `budgetID` _string_ | BudgetID is the identifier for the budget |  |  |
| `config` _object (keys:string, values:string)_ | Config contains additional configuration settings |  |  |
| `duration` _string_ | Duration specifies how long the key is valid |  |  |
| `enforcedParams` _string array_ | EnforcedParams lists parameters that must be included in requests |  |  |
| `guardrails` _string array_ | Guardrails defines guardrail settings |  |  |
| `key` _string_ | Key is the actual key value |  |  |
| `keyAlias` _string_ | KeyAlias is the user defined key alias |  | Required: \{\} <br /> |
| `maxBudget` _string_ | MaxBudget sets the maximum budget limit |  |  |
| `maxParallelRequests` _integer_ | MaxParallelRequests limits concurrent requests |  |  |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `modelMaxBudget` _object (keys:string, values:string)_ | ModelMaxBudget sets budget limits per model |  |  |
| `modelRPMLimit` _object (keys:string, values:integer)_ | ModelRPMLimit sets RPM limits per model |  |  |
| `modelTPMLimit` _object (keys:string, values:integer)_ | ModelTPMLimit sets TPM limits per model |  |  |
| `models` _string array_ | Models specifies which models can be used |  |  |
| `permissions` _object (keys:string, values:string)_ | Permissions defines key permissions |  |  |
| `rpmLimit` _integer_ | RPMLimit sets global RPM limit |  |  |
| `sendInviteEmail` _boolean_ | SendInviteEmail indicates whether to send an invite email |  |  |
| `softBudget` _string_ | SoftBudget sets a soft budget limit |  |  |
| `spend` _string_ | Spend tracks the current spend amount |  |  |
| `tags` _string array_ | Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license |  |  |
| `teamID` _string_ | TeamID identifies the team associated with the key |  |  |
| `tpmLimit` _integer_ | TPMLimit sets global TPM limit |  |  |
| `userID` _string_ | UserID identifies the user associated with the key |  |  |


#### VirtualKeyStatus



VirtualKeyStatus defines the observed state of VirtualKey



_Appears in:_
- [VirtualKey](#virtualkey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `aliases` _object (keys:string, values:string)_ | Aliases maps additional aliases for the key |  |  |
| `allowedCacheControls` _string array_ | AllowedCacheControls defines allowed cache control settings |  |  |
| `allowedRoutes` _string array_ | AllowedRoutes defines allowed API routes |  |  |
| `blocked` _boolean_ | Blocked indicates if the key is blocked |  |  |
| `budgetDuration` _string_ | BudgetDuration is the duration of the budget |  |  |
| `budgetID` _string_ | BudgetID is the identifier for the budget |  |  |
| `budgetResetAt` _string_ | BudgetResetAt is the date and time when the budget will reset |  |  |
| `config` _object (keys:string, values:string)_ | Config contains additional configuration settings |  |  |
| `createdAt` _string_ | CreatedAt is the date and time when the key was created |  |  |
| `createdBy` _string_ | CreatedBy tracks who created the key |  |  |
| `duration` _string_ | Duration specifies how long the key is valid |  |  |
| `enforcedParams` _string array_ | EnforcedParams lists parameters that must be included in requests |  |  |
| `expires` _string_ | Expires is the date and time when the key will expire |  |  |
| `guardrails` _string array_ | Guardrails defines guardrail settings |  |  |
| `keyAlias` _string_ | KeyAlias is the user defined key alias |  |  |
| `keyID` _string_ | KeyID is the generated ID of the key |  |  |
| `keyName` _string_ | KeyName is the redacted secret key |  |  |
| `keySecretRef` _string_ | KeySecretRef is the reference to the secret containing the key |  |  |
| `liteLLMBudgetTable` _string_ | LiteLLMBudgetTable is the budget table reference |  |  |
| `maxBudget` _string_ | MaxBudget is the maximum budget for the key |  |  |
| `maxParallelRequests` _integer_ | MaxParallelRequests limits concurrent requests |  |  |
| `models` _string array_ | Models specifies which models can be used |  |  |
| `permissions` _object (keys:string, values:string)_ | Permissions defines key permissions |  |  |
| `rpmLimit` _integer_ | RPMLimit sets global RPM limit |  |  |
| `spend` _string_ | Spend tracks the current spend amount |  |  |
| `tags` _string array_ | Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license |  |  |
| `teamID` _string_ | TeamID identifies the team associated with the key |  |  |
| `token` _string_ | Token contains the actual API key |  |  |
| `tokenID` _string_ | TokenID is the unique identifier for the token |  |  |
| `tpmLimit` _integer_ | TPMLimit sets global TPM limit |  |  |
| `updatedAt` _string_ | UpdatedAt is the date and time when the key was last updated |  |  |
| `updatedBy` _string_ | UpdatedBy tracks who last updated the key |  |  |
| `userID` _string_ | UserID identifies the user associated with the key |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v/#condition-v1-meta) array_ |  |  |  |


