# LiteLLM Operator Helm Chart

A Helm chart for deploying the LiteLLM Operator to Kubernetes clusters.

## Description

The LiteLLM Operator is a Kubernetes operator that manages LiteLLM resources including:
- Virtual Keys
- Users  
- Teams
- Team Member Associations

## Prerequisites

- Kubernetes 1.11.3+
- Helm 3.0+
- LiteLLM service running in the cluster

## Installation

### Add the Helm repository

```bash
helm repo add litellm-operator https://bbd.github.io/litellm-operator
helm repo update
```

### Install the chart

```bash
# Install with default values
helm install litellm-operator litellm-operator/litellm-operator

# Install with custom values
helm install litellm-operator litellm-operator/litellm-operator \
  --values values.yaml
```

### Install from local chart

```bash
# From the project root directory
helm install litellm-operator ./helm
```

## Configuration

The following table lists the configurable parameters of the litellm-operator chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.enabled` | Enable or disable the operator | `true` |
| `operator.image.repository` | Operator image repository | `ghcr.io/bbd/litellm-operator` |
| `operator.image.tag` | Operator image tag | `latest` |
| `operator.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `operator.replicas` | Number of operator replicas | `1` |
| `operator.resources.limits.cpu` | CPU limit | `500m` |
| `operator.resources.limits.memory` | Memory limit | `128Mi` |
| `operator.resources.requests.cpu` | CPU request | `10m` |
| `operator.resources.requests.memory` | Memory request | `64Mi` |
| `litellm.baseUrl` | LiteLLM service URL | `http://litellm:4000` |
| `litellm.masterKey.secretName` | Secret name for master key | `litellm-masterkey` |
| `litellm.masterKey.secretKey` | Secret key for master key | `masterkey` |
| `litellm.masterKey.create` | Create the master key secret | `false` |
| `litellm.masterKey.value` | Master key value (if create=true) | `""` |
| `namespace.create` | Create the namespace | `true` |
| `namespace.name` | Namespace name | `litellm` |
| `serviceAccount.create` | Create service account | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `crd.install` | Install CRDs | `true` |
| `operator.metrics.enabled` | Enable metrics endpoint | `true` |
| `operator.leaderElection.enabled` | Enable leader election | `true` |

### Example values.yaml

```yaml
operator:
  image:
    repository: ghcr.io/bbd/litellm-operator
    tag: "v0.1.0"
  
  resources:
    limits:
      cpu: 1000m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

litellm:
  baseUrl: "http://my-litellm-service:4000"
  masterKey:
    create: true
    value: "your-master-key-here"

namespace:
  name: "litellm-system"
```

## Usage

### Creating LiteLLM Resources

After installing the operator, you can create LiteLLM resources:

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: User
metadata:
  name: example-user
spec:
  # User specification
```

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: Team
metadata:
  name: example-team
spec:
  # Team specification
```

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: example-key
spec:
  # Virtual key specification
```

### Checking Operator Status

```bash
# Check operator deployment
kubectl get deployment -n litellm litellm-operator

# Check operator logs
kubectl logs -n litellm deployment/litellm-operator

# Check custom resources
kubectl get users,teams,virtualkeys -A
```

## Upgrading

```bash
# Upgrade with new values
helm upgrade litellm-operator litellm-operator/litellm-operator \
  --values values.yaml

# Upgrade from local chart
helm upgrade litellm-operator ./helm
```

## Uninstalling

```bash
# Uninstall the chart
helm uninstall litellm-operator

# Delete CRDs (optional)
kubectl delete crd users.auth.litellm.ai teams.auth.litellm.ai virtualkeys.auth.litellm.ai teammemberassociations.auth.litellm.ai
```

## Troubleshooting

### Operator not starting

1. Check if the LiteLLM service is accessible:
   ```bash
   kubectl exec -n litellm deployment/litellm-operator -- curl -s http://litellm:4000/health
   ```

2. Verify the master key secret exists:
   ```bash
   kubectl get secret -n litellm litellm-masterkey
   ```

3. Check operator logs:
   ```bash
   kubectl logs -n litellm deployment/litellm-operator
   ```

### CRDs not installed

If CRDs are not installed automatically:

```bash
# Install CRDs manually
kubectl apply -f helm/crds/
```

## Contributing

To contribute to this Helm chart:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test the chart locally
5. Submit a pull request

## License

This chart is licensed under the Apache 2.0 license. 