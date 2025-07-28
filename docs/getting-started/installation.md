# Installation

This guide will help you install the LiteLLM Operator in your Kubernetes cluster.

## Prerequisites

Before installing the operator, ensure you have:

- **Kubernetes v1.11.3+** - A running Kubernetes cluster
- **kubectl v1.11.3+** - Configured to access your cluster
- **Go v1.22.0+** - For building from source (optional)
- **Docker v17.03+** - For building container images (optional)

## Quick Installation

### 1. Install Custom Resource Definitions (CRDs)

First, install the custom resource definitions:

```bash
kubectl apply -f https://raw.githubusercontent.com/yourusername/litellm-operator/main/config/crd/bases/auth.litellm.ai_virtualkeys.yaml
kubectl apply -f https://raw.githubusercontent.com/yourusername/litellm-operator/main/config/crd/bases/auth.litellm.ai_users.yaml
kubectl apply -f https://raw.githubusercontent.com/yourusername/litellm-operator/main/config/crd/bases/auth.litellm.ai_teams.yaml
kubectl apply -f https://raw.githubusercontent.com/yourusername/litellm-operator/main/config/crd/bases/auth.litellm.ai_teammemberassociations.yaml
```

Or use the make target:

```bash
make install
```

### 2. Deploy the Operator

Deploy the operator to your cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/yourusername/litellm-operator/main/config/manager/manager.yaml
```

Or build and deploy from source:

```bash
make deploy IMG=<your-registry>/litellm-operator:latest
```

### 3. Verify Installation

Check that the operator is running:

```bash
kubectl get pods -n litellm-operator-system
```

You should see the operator pod in `Running` status.

## Installation from Source

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/litellm-operator.git
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

### RBAC Configuration

The operator requires appropriate RBAC permissions. If you encounter permission errors:

```bash
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole=cluster-admin \
  --user=$(kubectl config current-context)
```

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

- Check the [troubleshooting guide](../reference/troubleshooting.md)
- View operator logs: `kubectl logs -n litellm-operator-system deployment/litellm-operator-controller-manager`
- Submit an issue on [GitHub](https://github.com/bbdsoftware/litellm-operator/issues/new/choose)

## Next Steps

Once installed, proceed to the [Quick Start Guide](quickstart.md) to create your first resources. 