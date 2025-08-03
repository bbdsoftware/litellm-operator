# Quick Start

This guide will walk you through creating your first LiteLLM resources using the operator, focusing on setting up a LiteLLM Instance and creating a standalone virtual key.

## Prerequisites

- LiteLLM Operator [installed](installation.md) in your cluster
- PostgreSQL database accessible from your cluster
- Redis instance accessible from your cluster
- `kubectl` access to your cluster

## Step 1: Create Required Secrets

Before creating the LiteLLM Instance, you need to create Kubernetes secrets for database and Redis connections:

### Database Secret

```yaml
# postgres-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-postgres-host>
  password: <base64-encoded-postgres-password>
  username: <base64-encoded-postgres-username>
  dbname: <base64-encoded-postgres-database-name>
```

### Redis Secret

```yaml
# redis-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-redis-host>
  port: <base64-encoded-redis-port>
  password: <base64-encoded-redis-password>
```

Apply the secrets:

```bash
kubectl apply -f postgres-secret.yaml
kubectl apply -f redis-secret.yaml
```

## Step 2: Create a LiteLLM Instance

Create the core LiteLLM Instance that will manage your proxy server:

```yaml
# litellm-instance.yaml
apiVersion: litellm.litellm.ai/v1alpha1
kind: LiteLLMInstance
metadata:
  name: litellm-example
  namespace: litellm
spec:
  redisSecretRef:
    nameRef: redis-secret
    keys:
      hostSecret: host
      portSecret: 6379
      passwordSecret: password
  databaseSecretRef:
    nameRef: postgres-secret
    keys:
      hostSecret: host
      passwordSecret: password
      usernameSecret: username
      dbnameSecret: dbname
```

Apply the LiteLLM Instance:

```bash
kubectl apply -f litellm-instance.yaml
```

Verify the instance is created and running:

```bash
kubectl get litellminstances
kubectl describe litellminstance litellm-example
```

## Step 3: Create a Standalone Virtual Key

Create a virtual key for API access to your LiteLLM proxy:

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
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

Apply the virtual key:

```bash
kubectl apply -f virtualkey-example.yaml
```

Verify the virtual key was created:

```bash
kubectl get virtualkeys
```

## Step 4: Verify Everything

Check that all resources are created and ready:

```bash
# Check all resources
kubectl get litellminstances,virtualkeys

# Get detailed status
kubectl describe litellminstance litellm-example
kubectl describe virtualkey example-service
```

## Using the Virtual Key

Once created, the virtual key can be retrieved from the resource status and used to authenticate with the LiteLLM proxy:

### Get the Virtual Key Value

```bash
# Get the virtual key value
kubectl get virtualkey example-service -o jsonpath='{.status.keyValue}'
```

### Make API Calls

Use the key in your API calls:

```bash
# Set the key as a variable
KEY=$(kubectl get virtualkey example-service -o jsonpath='{.status.keyValue}')

# Make an API call
curl -X POST "http://your-litellm-endpoint/chat/completions" \
  -H "Authorization: Bearer $KEY" \
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



## Complete Example File

You can also create all resources in a single file:

```yaml
# complete-example.yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-postgres-host>
  password: <base64-encoded-postgres-password>
  username: <base64-encoded-postgres-username>
  dbname: <base64-encoded-postgres-database-name>
---
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-redis-host>
  port: <base64-encoded-redis-port>
  password: <base64-encoded-redis-password>
---
apiVersion: litellm.litellm.ai/v1alpha1
kind: LiteLLMInstance
metadata:
  name: litellm-example
  namespace: litellm
spec:
  redisSecretRef:
    nameRef: redis-secret
    keys:
      hostSecret: host
      portSecret: 6379
      passwordSecret: password
  databaseSecretRef:
    nameRef: postgres-secret
    keys:
      hostSecret: host
      passwordSecret: password
      usernameSecret: username
      dbnameSecret: dbname
---
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
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

Apply everything at once:

```bash
kubectl apply -f complete-example.yaml
```

## Next Steps

- Learn more about [LiteLLM Instances](../user-guide/litellm-instances.md)
- Explore [Virtual Keys](../user-guide/virtual-keys.md)
- Understand [User Management](../user-guide/users.md) and [Team Management](../user-guide/teams.md)
- Check out [sample configurations](https://github.com/bbdsoftware/litellm-operator/tree/main/config/samples)

## Cleanup

To remove all resources created in this guide:

```bash
kubectl delete virtualkey example-service
kubectl delete litellminstance litellm-example
kubectl delete secret redis-secret
kubectl delete secret postgres-secret
``` 
