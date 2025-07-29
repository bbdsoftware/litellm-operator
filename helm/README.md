# LiteLLM Operator Helm Chart

This Helm chart deploys the LiteLLM Operator to a Kubernetes cluster.

## Installation

The chart is available as an OCI image in GitHub Packages. Install it directly using:

```bash
# Login to GitHub Container Registry (if not already logged in)
helm registry login ghcr.io

# Install the chart
helm install litellm-operator oci://ghcr.io/bbd/charts/litellm-operator --version <VERSION>
```

### Example Installation

```bash
# Install the latest version
helm install litellm-operator oci://ghcr.io/bbd/charts/litellm-operator

# Install a specific version
helm install litellm-operator oci://ghcr.io/bbd/charts/litellm-operator --version 1.2.3

# Install with custom values
helm install litellm-operator oci://ghcr.io/bbd/charts/litellm-operator \
  --set operator.replicas=3 \
  --set litellm.baseUrl=http://my-litellm:4000
```

## Configuration

See the [values.yaml](values.yaml) file for all available configuration options.

## Usage

After installation, the LiteLLM Operator will be available in your cluster and you can create custom resources like:

- Teams
- Users
- Virtual Keys
- Team Member Associations

For more information, see the [LiteLLM Operator documentation](https://github.com/bbd/litellm-operator).
