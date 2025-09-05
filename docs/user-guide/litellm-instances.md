# LiteLLM Instances

LiteLLM Instances are the core resources that deploy and manage LiteLLM proxy servers in your Kubernetes cluster. They provide the infrastructure for authentication, model routing, and API management.

## Overview

LiteLLM Instance resources in the LiteLLM Operator provide:

- **Proxy Deployment** - Deploy LiteLLM proxy servers in Kubernetes
- **Database Integration** - Connect to PostgreSQL databases for user management
- **Redis Caching** - Configure Redis for session and cache management
- **Network Access** - Set up ingress and gateway configurations
- **Resource Management** - Manage deployments, services, and secrets

## Creating LiteLLM Instances

### Basic Instance

```yaml
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

### Advanced Instance with Ingress and Gateway

```yaml
apiVersion: litellm.litellm.ai/v1alpha1
kind: LiteLLMInstance
metadata:
  name: litellm-production
  namespace: litellm
spec:
  image: "ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1"
  masterKey: "sk-1234567890abcdef"
  replicas: 3
  redisSecretRef:
    nameRef: redis-production
    keys:
      hostSecret: host
      portSecret: port
      passwordSecret: password
  databaseSecretRef:
    nameRef: postgres-production
    keys:
      hostSecret: host
      passwordSecret: password
      usernameSecret: username
      dbnameSecret: dbname
  ingress:
    enabled: true
    host: "api.litellm.example.com"
  gateway:
    enabled: true
    host: "gateway.litellm.example.com"
```

### High Availability Instance

```yaml
apiVersion: litellm.litellm.ai/v1alpha1
kind: LiteLLMInstance
metadata:
  name: litellm-ha
  namespace: litellm
spec:
  replicas: 5
  redisSecretRef:
    nameRef: redis-ha
    keys:
      hostSecret: host
      portSecret: port
      passwordSecret: password
  databaseSecretRef:
    nameRef: postgres-ha
    keys:
      hostSecret: host
      passwordSecret: password
      usernameSecret: username
      dbnameSecret: dbname
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `image` | string | LiteLLM Docker image | No (default: ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1) |
| `masterKey` | string | Master API key for the instance | No |
| `databaseSecretRef` | object | PostgreSQL database configuration | No |
| `redisSecretRef` | object | Redis cache configuration | No |
| `ingress` | object | Kubernetes ingress configuration | No |
| `gateway` | object | Gateway configuration | No |
| `replicas` | integer | Number of replicas for the LiteLLM deployment | No (default: 1) |

### Database Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `nameRef` | string | Name of the Kubernetes secret containing database credentials | Yes |
| `keys.hostSecret` | string | Secret key containing database host | Yes |
| `keys.passwordSecret` | string | Secret key containing database password | Yes |
| `keys.usernameSecret` | string | Secret key containing database username | Yes |
| `keys.dbnameSecret` | string | Secret key containing database name | Yes |

### Redis Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `nameRef` | string | Name of the Kubernetes secret containing Redis credentials | Yes |
| `keys.hostSecret` | string | Secret key containing Redis host | Yes |
| `keys.portSecret` | int | Secret key containing Redis port | Yes |
| `keys.passwordSecret` | string | Secret key containing Redis password | Yes |

### Ingress Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `enabled` | boolean | Whether to enable ingress | No (default: false) |
| `host` | string | Hostname for the ingress | Yes (if enabled) |

### Gateway Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `enabled` | boolean | Whether to enable gateway | No (default: false) |
| `host` | string | Hostname for the gateway | Yes (if enabled) |

## Managing LiteLLM Instances

### List Instances

```bash
kubectl get litellminstances
```

### Get Instance Details

```bash
kubectl describe litellminstance litellm-example
```

### Check Instance Status

```bash
kubectl get litellminstance litellm-example -o yaml
```

### Update Instance Configuration

```bash
kubectl patch litellminstance litellm-example --type='merge' -p='{"spec":{"masterKey":"sk-new-key"}}'
```

### Delete an Instance

```bash
kubectl delete litellminstance litellm-example
```

### Scaling Instances

You can scale LiteLLM instances by updating the replicas field:

```bash
# Scale to 3 replicas
kubectl patch litellminstance litellm-example --type='merge' -p='{"spec":{"replicas":3}}'

# Scale to 1 replica
kubectl patch litellminstance litellm-example --type='merge' -p='{"spec":{"replicas":1}}'
```

## Status Information

The LiteLLM Instance status provides information about the deployment:

```bash
kubectl get litellminstance litellm-example -o jsonpath='{.status}'
```

### Status Fields

- `observedGeneration` - Generation of the spec that was last observed
- `lastUpdated` - Timestamp of last status update
- `configMapCreated` - Whether the ConfigMap was created
- `secretCreated` - Whether the Secret was created
- `deploymentCreated` - Whether the Deployment was created
- `serviceCreated` - Whether the Service was created
- `ingressCreated` - Whether the Ingress was created
- `conditions` - Array of condition objects

## Prerequisites

### Required Secrets

Before creating a LiteLLM Instance, ensure you have the necessary Kubernetes secrets:

#### Database Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-host>
  password: <base64-encoded-password>
  username: <base64-encoded-username>
  dbname: <base64-encoded-database-name>
```

#### Redis Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
  namespace: litellm
type: Opaque
data:
  host: <base64-encoded-host>
  port: <base64-encoded-port>
  password: <base64-encoded-password>
```

### Database Setup

Ensure your PostgreSQL database is accessible and contains the necessary tables for LiteLLM:

- User management tables
- Team management tables
- Virtual key tables
- Usage tracking tables

## Best Practices

### Security
- Use strong master keys for production instances
- Store sensitive credentials in Kubernetes secrets
- Enable TLS for database and Redis connections
- Use network policies to restrict access

### Performance
- Use dedicated Redis instances for caching
- Configure appropriate resource limits
- Monitor database connection pools
- Set up proper backup strategies
- Scale replicas based on load requirements
- Consider horizontal pod autoscaling for dynamic workloads

### Monitoring
- Enable ingress for external access
- Set up monitoring and alerting
- Track resource usage and costs
- Monitor API usage patterns

### Organisation
- Use descriptive names for instances
- Organise instances by environment (dev, staging, prod)
- Document configuration changes
- Maintain consistent naming conventions

## Troubleshooting

### Common Issues

**Instance Not Starting**
- Check that all required secrets exist
- Verify database connectivity
- Ensure Redis is accessible
- Check resource limits and quotas

**Database Connection Issues**
- Verify secret keys match the configuration
- Check database host and port
- Ensure database user has proper permissions
- Verify network policies allow connections

**Redis Connection Issues**
- Check Redis host and port configuration
- Verify Redis password in secret
- Ensure Redis is running and accessible
- Check network connectivity

**Ingress Not Working**
- Verify ingress controller is installed
- Check hostname configuration
- Ensure TLS certificates are valid
- Verify ingress annotations are correct

## Integration with Other Resources

LiteLLM Instances are referenced by other resources:

### Users, Teams, and Virtual Keys

All authentication resources reference the LiteLLM Instance:

```yaml
connectionRef:
  instanceRef:
    name: litellm-example
    namespace: litellm
```

### Complete Example

```yaml
# 1. Create the LiteLLM Instance
apiVersion: litellm.litellm.ai/v1alpha1
kind: LiteLLMInstance
metadata:
  name: litellm-example
  namespace: litellm
spec:
  replicas: 2
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
# 2. Create a User that references the instance
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
```

## Next Steps

- Learn about [Users](users.md), [Teams](teams.md), and [Virtual Keys](virtual-keys.md)
- Review [security best practices](../community/security.md) 
