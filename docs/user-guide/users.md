# Users

Users represent individual accounts in your LiteLLM system. They provide the foundation for authentication, authorization, and budget management.

## Overview

User resources in the LiteLLM Operator provide:

- **Identity Management** - Define user accounts and roles
- **Budget Control** - Set spending limits per user
- **Role-Based Access** - Assign different permission levels
- **Usage Tracking** - Monitor individual user consumption
- **Automatic Key Creation** - Generate virtual keys automatically

## Creating Users

### Basic User

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: alice
spec:
  userEmail: "alice@example.com"
  userAlias: "alice"
  userRole: "customer"
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
```

### Admin User

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: admin-user
spec:
  userEmail: "admin@example.com"
  userAlias: "admin"
  userRole: "admin"
  keyAlias: "admin-key"
  autoCreateKey: true
  models:
    - "gpt-4o"
    - "claude-3-sonnet"
  maxBudget: "1000"
  budgetDuration: 30d
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `userEmail` | string | User's email address | Yes |
| `userAlias` | string | User alias/username | Yes |
| `userRole` | string | User role (customer, admin) | Yes |
| `keyAlias` | string | Alias for the virtual key | Yes |
| `autoCreateKey` | boolean | Automatically create virtual key | Yes |
| `models` | []string | Allowed models for this user | No |
| `maxBudget` | string | Maximum spend limit in dollars | Yes |
| `budgetDuration` | string | Budget duration (e.g., "1h", "30d") | Yes |
| `connectionRef` | object | Reference to LiteLLM instance | Yes |

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
kubectl patch user alice --type='merge' -p='{"spec":{"maxBudget":"200"}}'
```

### Delete a User

```bash
kubectl delete user alice
```

## User Roles

### Customer User
- Can create virtual keys
- Limited to personal budget
- Cannot manage other users
- Access to specified models only

### Admin User
- Can manage all users and teams
- Can create system-wide policies
- Access to admin endpoints
- Full model access

## Best Practices

- Use email addresses as user identifiers for consistency
- Set appropriate budget limits based on usage patterns
- Use meaningful key aliases for easy identification
- Enable autoCreateKey for seamless user onboarding
- Set reasonable budget durations to prevent overspending
- Regularly review and update user permissions

## Next Steps

- Learn about [Teams](teams.md) and [Team Member Associations](team-member-associations.md)
- Create [Virtual Keys](virtual-keys.md) for users
- Set up user monitoring and alerts 