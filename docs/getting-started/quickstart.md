# Quick Start

This guide will walk you through creating your first LiteLLM resources using the operator.

## Prerequisites

- LiteLLM Operator [installed](installation.md) in your cluster
- LiteLLM service running in your cluster
- `kubectl` access to your cluster

## Step 1: Create a User

First, let's create a user resource:

```yaml
# user-example.yaml
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
  namespace: default
spec:
  teamId: "ai-team"
  maxBudget: 500.0
  tpmLimit: 1000
  rpmLimit: 100
```

Apply the team:

```bash
kubectl apply -f team-example.yaml
```

## Step 3: Associate User with Team

Create a team member association:

```yaml
# association-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: TeamMemberAssociation
metadata:
  name: alice-ai-team
  namespace: default
spec:
  userId: "alice@example.com"
  teamId: "ai-team"
  role: "member"
```

Apply the association:

```bash
kubectl apply -f association-example.yaml
```

## Step 4: Create a Virtual Key

Finally, create a virtual key for API access:

```yaml
# virtualkey-example.yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: alice-key
  namespace: default
spec:
  userId: "alice@example.com"
  maxBudget: 50.0
  models: ["gpt-3.5-turbo", "gpt-4"]
  aliases: 
    gpt-3.5-turbo: "azure/gpt-35-turbo"
  duration: "30d"
```

Apply the virtual key:

```bash
kubectl apply -f virtualkey-example.yaml
```

## Step 5: Verify Everything

Check that all resources are created and ready:

```bash
# Check all resources
kubectl get users,teams,teammemberassociations,virtualkeys

# Get detailed status
kubectl describe user alice
kubectl describe team ai-team
kubectl describe virtualkey alice-key
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
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
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
kubectl delete virtualkey alice-key
kubectl delete teammemberassociation alice-ai-team  
kubectl delete team ai-team
kubectl delete user alice
``` 