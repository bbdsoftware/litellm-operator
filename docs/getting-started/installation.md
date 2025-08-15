# Installation

This guide will help you install the LiteLLM Operator in your Kubernetes cluster.

## Prerequisites

Before installing the operator, ensure you have:

- **Kubernetes v1.11.3+** - A running Kubernetes cluster
- **kubectl v1.11.3+** - Configured to access your cluster
- **Go v1.22.0+** - For building from source (optional)
- **Docker v17.03+** - For building container images (optional)

## Quick Installation

### 1. Install the operator

#### Helm

```bash
helm install --namespace litellm litellm-operator oci://ghcr.io/bbdsoftware/charts/litellm-operator:<version>
```

#### Kustomize

```bash
kubectl --namespace litellm apply -k config/default
```

### 2. Verify Installation

Check that the operator is running:

```bash
kubectl get pods --namespace litellm
```

You should see the operator pod in `Running` status.

## Installation from Source

### 1. Clone the Repository

```bash
git clone https://github.com/bbdsoftware/litellm-operator.git
cd litellm-operator
```

### 2. Build and Push Image

```bash
make docker-build docker-push IMG=<your-registry>/litellm-operator:tag
```

### 3. Install CRDs

```bash
make install
```

### 4. Deploy Operator

```bash
make deploy IMG=<your-registry>/litellm-operator:tag
```

## Configuration

### Environment Variables

The operator supports the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LITELLM_BASE_URL` | Base URL for LiteLLM API | `http://litellm:4000` |
| `LITELLM_API_KEY` | API key for LiteLLM authentication | Required |
| `METRICS_BIND_ADDRESS` | Address for metrics server | `:8080` |
| `HEALTH_PROBE_BIND_ADDRESS` | Address for health probes | `:8081` |

## Troubleshooting

### Common Issues

**Permission Denied Errors**
- Ensure you have cluster-admin privileges
- Check RBAC configuration

**Image Pull Errors**
- Verify the image registry is accessible
- Check image tag and repository URL

**CRD Installation Failures**
- Ensure you have permission to create CRDs
- Check for existing CRDs that might conflict

### Getting Help

- View operator logs: `kubectl logs -n litellm deployment/litellm-operator-controller-manager`
- Submit an issue on [GitHub](https://github.com/bbdsoftware/litellm-operator/issues/new/choose)

## Next Steps

Once installed, proceed to the [Quick Start Guide](quickstart.md) to create your first resources. 
