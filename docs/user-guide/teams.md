# Teams

Teams provide a way to group users and manage shared resources, permissions, and model access in your LiteLLM system.

## Overview

Team resources in the LiteLLM Operator provide:

- **Group Management** - Organise users into logical groups
- **Shared Model Access** - Define which models team members can use
- **Resource Sharing** - Share LiteLLM instances across team members
- **Permission Management** - Control team-level access and permissions

## Creating Teams

### Basic Team

```yaml
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
```

### Advanced Team with Multiple Models

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: Team
metadata:
  name: research-team
spec:
  teamAlias: research-team
  models:
    - gpt-4o
    - claude-3-sonnet
    - gemini-pro
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `teamAlias` | string | Unique team identifier | Yes |
| `models` | []string | Models available to team members | No |
| `connectionRef` | object | Reference to LiteLLM instance | Yes |

## Managing Teams

### List Teams

```bash
kubectl get teams
```

### Get Team Details

```bash
kubectl describe team ai-team
```

### Update Team Models

```bash
kubectl patch team ai-team --type='merge' -p='{"spec":{"models":["gpt-4o","claude-3-sonnet"]}}'
```

### Delete a Team

```bash
kubectl delete team ai-team
```

## Team Member Associations

Teams work in conjunction with Team Member Associations to manage user membership and roles within teams. See [Team Member Associations](team-member-associations.md) for detailed information.

### Adding Users to Teams

Users are added to teams through Team Member Association resources:

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
```

## Best Practices

- Use descriptive team aliases that reflect the team's purpose
- Limit model access to only what team members need
- Regularly review team membership and permissions
- Use consistent naming conventions across teams
- Consider team structure when planning user organisation

## Next Steps

- Learn about [Team Member Associations](team-member-associations.md)
- Understand [Users](users.md) and their roles
- Create [Virtual Keys](virtual-keys.md) for team members
