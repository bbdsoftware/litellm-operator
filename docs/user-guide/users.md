# Users

Users represent individual accounts in your LiteLLM system. They provide the foundation for authentication, authorization, and budget management.

## Overview

User resources in the LiteLLM Operator provide:

- **Identity Management** - Define user accounts and roles
- **Budget Control** - Set spending limits per user
- **Role-Based Access** - Assign different permission levels
- **Usage Tracking** - Monitor individual user consumption

## Creating Users

### Basic User

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: alice
  namespace: default
spec:
  userId: "alice@example.com"
  userRole: "user"
  maxBudget: 100.0
```

### Admin User

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: admin-user
  namespace: default
spec:
  userId: "admin@example.com"
  userRole: "admin"
  maxBudget: 1000.0
  metadata:
    department: "engineering"
    team: "platform"
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `userId` | string | Unique user identifier | Yes |
| `userRole` | string | User role (user, admin) | Yes |
| `maxBudget` | float | Maximum spend limit in dollars | Yes |
| `metadata` | map[string]string | Custom metadata | No |

## Managing Users

### List Users

```bash
kubectl get users
```

### Get User Details

```bash
kubectl describe user alice
```

### Update User Budget

```bash
kubectl patch user alice --type='merge' -p='{"spec":{"maxBudget":200.0}}'
```

### Delete a User

```bash
kubectl delete user alice
```

## User Roles

### Standard User
- Can create virtual keys
- Limited to personal budget
- Cannot manage other users

### Admin User
- Can manage all users and teams
- Can create system-wide policies
- Access to admin endpoints

## Best Practices

- Use email addresses as user IDs for consistency
- Set appropriate budget limits based on usage patterns
- Regularly review and update user permissions
- Use metadata for organizational tracking

## Next Steps

- Learn about [Teams](teams.md) and [Team Member Associations](team-member-associations.md)
- Create [Virtual Keys](virtual-keys.md) for users
- Set up user monitoring and alerts 