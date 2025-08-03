# Quick Start

This guide will walk you through creating your first LiteLLM resources using the operator.

## Prerequisites

- LiteLLM Operator [installed](installation.md) in your cluster
- LiteLLM service running in your cluster
- `kubectl` access to your cluster

## Step 1: Create a User

First, let's create a user resource:

!!! note 
    Creating a user will automatically create a virtual key for the user unless `autoCreateKey` is set to `false`.

```yaml
# user-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: alice
spec:
  userEmail: alice@example.com
  userAlias: alice
  userRole: customer
  keyAlias: alice-key
  autoCreateKey: true
  models:
    - gpt-4o
  maxBudget: "10"
  budgetDuration: 1h
```

Apply the user:

```bash
kubectl apply -f user-example.yaml
```

Verify the user was created:

```bash
kubectl get users
```

## Step 2: Create a Team

Next, create a team:

```yaml
# team-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: Team
metadata:
  name: ai-team
spec:
  teamAlias: ai-team
  models:
    - gpt-4o
```

Apply the team:

```bash
kubectl apply -f team-example.yaml
```

Verify the team was created:

```bash
kubectl get teams
```

## Step 3: Associate User with Team

Create a team member association:

```yaml
# association-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: TeamMemberAssociation
metadata:
  name: alice-ai-team
spec:
  role: member
  teamAlias: ai-team
  userEmail: alice@example.com
```

Apply the association:

```bash
kubectl apply -f association-example.yaml
```

Verify the association was created:

```bash
kubectl get teammemberassociations
```

## Step 4: Create a Virtual Key (optional)

Create a virtual key that is not associated with a user:

```yaml
# virtualkey-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: example-service
spec:
  keyAlias: example-service
  models:
    - gpt-4o
  maxBudget: "10"
  budgetDuration: 1h
```

Apply the virtual key:

```bash
kubectl apply -f virtualkey-example.yaml
```

Verify the virtual key was created:

```bash
kubectl get virtualkeys
```

## Step 5: Verify Everything

Check that all resources are created and ready:

```bash
# Check all resources
kubectl get users,teams,teammemberassociations,virtualkeys

# Get detailed status
kubectl describe user alice
kubectl describe team ai-team
kubectl describe virtualkey example-service
```

## Using the Virtual Key

Once created, the virtual key can be retrieved from the resource status and used to authenticate with the LiteLLM proxy:

```bash
# Get the virtual key value
kubectl get virtualkey alice-key -o jsonpath='{.status.keyValue}'
```

Use this key in your API calls:

```bash
curl -X POST "http://your-litellm-endpoint/chat/completions" \
  -H "Authorization: Bearer <virtual-key-value>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful software engineer. Guide the user through the solution step by step."
      },
      {
        "role": "user",
        "content": "How can I create a Kubernetes operator?"
      }
    ]
  }'
```

## Next Steps

- Learn more about [Virtual Keys](../user-guide/virtual-keys.md)
- Explore [Team Management](../user-guide/teams.md)
- Check out the [User Guide](../user-guide/users.md)
- View [sample configurations](https://github.com/yourusername/litellm-operator/tree/main/config/samples)

## Cleanup

To remove the resources created in this guide:

```bash
kubectl delete virtualkey example-service
kubectl delete teammemberassociation alice-ai-team  
kubectl delete team ai-team
kubectl delete user alice
``` 
