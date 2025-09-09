# Team Member Associations

Team Member Associations define the relationship between users and teams, including roles and permissions within each team.

## Overview

Team Member Association resources in the LiteLLM Operator provide:

- **User-Team Linking** - Connect users to specific teams
- **Role Management** - Define user roles within teams
- **Permission Control** - Manage team-specific access rights
- **Team Organisation** - Structure team hierarchies and responsibilities

## Creating Team Member Associations

### Basic Association

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: TeamMemberAssociation
metadata:
  name: alice-ai-team
spec:
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
  role: admin
  teamAlias: ai-team
  userEmail: alice@example.com
  teamRef:
    name: ai-team
    namespace: litellm
  userRef:
    name: alice
    namespace: litellm
```

### Member Association

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: TeamMemberAssociation
metadata:
  name: bob-ai-team
spec:
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
  role: member
  teamAlias: ai-team
  userEmail: bob@example.com
  teamRef:
    name: ai-team
    namespace: litellm
  userRef:
    name: bob
    namespace: litellm
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `connectionRef` | object | Reference to LiteLLM instance | Yes |
| `role` | string | User role within the team (admin, member) | Yes |
| `teamAlias` | string | Team identifier to associate with | Yes |
| `userEmail` | string | User's email address | Yes |
| `teamRef` | object | Reference to an existing `Team` resource  | yes |
| `userRef` | object | Reference to an existing `User` resource  | yes |

## Managing Team Member Associations

### List Associations

```bash
kubectl get teammemberassociations
```

### Get Association Details

```bash
kubectl describe teammemberassociation alice-ai-team
```

### Update User Role

```bash
kubectl patch teammemberassociation alice-ai-team --type='merge' -p='{"spec":{"role":"member"}}'
```

### Remove User from Team

```bash
kubectl delete teammemberassociation alice-ai-team
```

## Team Roles

### Admin Role
- Can manage team settings and membership
- Can modify team model access
- Can view all team member activities
- Can create and manage team resources

### Member Role
- Can access team resources and models
- Can view team information
- Cannot modify team settings
- Limited to personal permissions

## Working with Teams and Users

### Prerequisites

Before creating a Team Member Association, ensure:

1. **Team exists**: The team referenced by `teamAlias` must already exist
2. **User exists**: The user referenced by `userEmail` must already exist
3. **LiteLLM Instance**: The referenced LiteLLM instance must be available

If you use `teamRef` or `userRef`, the referenced `Team` and `User` resources must exist in the cluster (the namespace may be omitted to use the same namespace as the association).

### Complete Workflow Example

```yaml
# 1. Create the team
apiVersion: auth.litellm.ai/v1alpha1
kind: Team
metadata:
  name: ai-team
spec:
  teamAlias: ai-team
  models:
    - gpt-4o
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
---
# 2. Create the user
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: alice
spec:
  userEmail: "alice@example.com"
  userAlias: "alice"
  userRole: "internal_user_viewer"
  keyAlias: "alice-key"
  autoCreateKey: true
  models:
    - "gpt-4o"
  maxBudget: "10"
  budgetDuration: 1h
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
---
# 3. Associate user with team
apiVersion: auth.litellm.ai/v1alpha1
kind: TeamMemberAssociation
metadata:
  name: alice-ai-team
spec:
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
  role: admin
  teamAlias: ai-team
  userEmail: alice@example.com
```

## Best Practices

- Use descriptive names for associations that include both user and team
- Start with member roles and promote to admin only when necessary
- Regularly review team membership and roles
- Ensure team aliases match exactly between Team and TeamMemberAssociation resources
- Use consistent email addresses across User and TeamMemberAssociation resources

## Troubleshooting

### Common Issues

**Association Not Working**
- Verify the team exists with the correct `teamAlias`
- Check that the user exists with the correct `userEmail`
- Ensure the LiteLLM instance is running and accessible

**Role Permissions**
- Admin roles have full team management capabilities
- Member roles have limited access to team resources
- Verify role spelling (admin, member)

## Next Steps

- Learn about [Teams](teams.md) and [Users](users.md)
- Understand [Virtual Keys](virtual-keys.md) and their usage
- Set up monitoring and access control
