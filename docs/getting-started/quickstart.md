# Quick Start

This guide will walk you through creating your first LiteLLM resources using the operator, focusing on setting up a LiteLLM Instance and creating a standalone virtual key.

## Prerequisites

- LiteLLM Operator [installed](installation.md) in your cluster
- PostgreSQL database accessible from your cluster
- Redis instance accessible from your cluster
- `kubectl` access to your cluster

## Optional: Deploy PostgreSQL using CloudNativePG Operator

If you don't have an existing PostgreSQL database, you can deploy one using the CloudNativePG operator. This provides a cloud-native, production-ready PostgreSQL solution for Kubernetes.

### Install CloudNativePG Operator

```bash
# Install the CloudNativePG operator
kubectl apply -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.21/releases/cnpg-1.21.0.yaml
```

### Deploy PostgreSQL Cluster

Create a PostgreSQL cluster for LiteLLM:

```yaml
# postgres-cluster.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: litellm-postgres
  namespace: litellm
spec:
  instances: 1

  postgresql:
    parameters:
      max_connections: '200'
      shared_buffers: '256MB'
      effective_cache_size: '1GB'

  bootstrap:
    initdb:
      database: litellm
      owner: litellm
      secret:
        name: litellm-postgres-credentials

  storage:
    size: 10Gi
    storageClass: 'standard' # Adjust based on your cluster's storage classes

  resources:
    requests:
      memory: '512Mi'
      cpu: '500m'
    limits:
      memory: '1Gi'
      cpu: '1000m'

---
apiVersion: v1
kind: Secret
metadata:
  name: litellm-postgres-credentials
  namespace: litellm
type: Opaque
data:
  username: bGl0ZWxsbQ== # litellm
  password: cGFzc3dvcmQxMjM= # password123 - Change this!
```

Apply the PostgreSQL cluster:

```bash
# Create the litellm namespace if it doesn't exist
kubectl create namespace litellm

# Deploy PostgreSQL
kubectl apply -f postgres-cluster.yaml
```

Wait for the cluster to be ready:

```bash
# Check cluster status
kubectl get cluster -n litellm
kubectl get pods -n litellm

# Wait for all pods to be ready (this may take a few minutes)
kubectl wait --for=condition=Ready cluster/litellm-postgres -n litellm --timeout=300s
```

### Get PostgreSQL Connection Details

Once the cluster is ready, get the connection details:

```bash
# Get the PostgreSQL service name
kubectl get svc -n litellm | grep litellm-postgres

# The service will be named: litellm-postgres-rw (for read-write)
# The connection details will be:
# Host: litellm-postgres-rw.litellm.svc.cluster.local
# Port: 5432
# Database: litellm
# Username: litellm
# Password: password123 (or whatever you set in the secret)
```

## Optional: Deploy Redis

The below example deploys a single instance of Redis

```yaml
#redis-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:8.2.0 #For more versions, see https://hub.docker.com/_/redis
          ports:
            - containerPort: 6379
          volumeMounts:
            - name: redis-storage
              mountPath: /data
          env:
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-secret
                  key: redis-password
      volumes:
        - name: redis-storage
          persistentVolumeClaim:
            claimName: redis-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
spec:
  selector:
    app: redis
  ports:
    - port: 6379
      targetPort: 6379
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
type: Opaque
data:
  # Base64 encoded password (echo -n 'password123' | base64). Change this!
  redis-password: cGFzc3dvcmQxMjM=
```

## Step 1: Create Required Secrets

Before creating the LiteLLM Instance, you need to create Kubernetes secrets for database and Redis connections:

### Database Secret

If you deployed PostgreSQL using the CloudNativePG operator from the previous section, use these values:

```yaml
# postgres-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: litellm
type: Opaque
data:
  host: bGl0ZWxsbS1wb3N0Z3Jlcy1ydy5saXRlbGxtLnN2Yy5jbHVzdGVyLmxvY2Fs # litellm-postgres-rw.litellm.svc.cluster.local
  password: cGFzc3dvcmQxMjM= # password123 (should match what you set in postgres-cluster.yaml)
  username: bGl0ZWxsbQ== # litellm
  dbname: bGl0ZWxsbQ== # litellm
```

For an external PostgreSQL database, use your own connection details:

```yaml
# postgres-secret.yaml (external database example)
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
  password: <base64-encoded-redis-password> #should match the password in redis-deployment.yaml
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
      portSecret: port
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
  maxBudget: '10'
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
KEY_SECRET=$(kubectl get virtualkey example-service -o jsonpath='{.status.keySecretRef}')

KEY=$(kubectl get secret $KEY_SECRET -o jsonpath='{.data.key}' | base64 -d)
```

### Make API Calls

Use the key in your API calls:

```bash
# Set the key as a variable
KEY_SECRET=$(kubectl get virtualkey example-service -o jsonpath='{.status.keySecretRef}')

KEY=$(kubectl get secret $KEY_SECRET -o jsonpath='{.data.key}' | base64 -d)

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
      portSecret: port
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
  maxBudget: '10'
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
