# LiteLLM Operator Helm Repository

This is the Helm repository for the LiteLLM Operator.

## 🎯 **Complete Helm Chart Structure**

### **Core Chart Files:**
- **`helm/Chart.yaml`** - Chart metadata and versioning
- **`helm/values.yaml`** - All configurable parameters with sensible defaults
- **`helm/templates/`** - All Kubernetes resource templates
- **`helm/crds/`** - Custom Resource Definitions (copied from your config)

### **Key Features:**

1. ** Fully Configurable** - All aspects of the operator can be customized via values.yaml
2. **🛡️ Security-First** - Proper security contexts, RBAC, and secret management
3. **📊 Metrics Ready** - Built-in Prometheus metrics support
4. ** Production Ready** - Health checks, leader election, resource limits
5. ** CRD Management** - Automatic CRD installation and cleanup options

### **🚀 Automated Publishing:**

1. **GitHub Workflow** (`.github/workflows/helm.yml`) that:
   - Lints and tests the chart on every PR
   - Automatically publishes to OCI registry (ghcr.io) on releases
   - Updates chart version to match release tags

2. **PR Validation Workflow** (`.github/workflows/helm-pr-validation.yml`) that:
   - Validates Helm chart changes in PRs
   - Checks version consistency
   - Provides feedback on PRs

3. **Makefile Targets** for local development:
   ```bash
   make helm-lint      # Lint the chart
   make helm-package   # Package the chart
   make helm-install   # Install locally
   make helm-test      # Test the chart
   ```

### ** Documentation:**

- **Comprehensive README** with installation instructions
- **Configuration table** with all available options
- **Troubleshooting guide** for common issues
- **Usage examples** for creating LiteLLM resources

### ** Staying Up-to-Date:**

The chart automatically stays current because:

1. **Release-triggered publishing** - New chart versions are published with each GitHub release
2. **Version synchronization** - Chart version automatically matches your operator version
3. **CRD updates** - CRDs are automatically copied from your config directory
4. **Image tag updates** - Chart automatically uses the correct image tag for each release

### **📦 Installation Methods:**

Users can install your operator using OCI (recommended):

```bash
# From OCI registry (recommended)
helm registry login ghcr.io
helm install litellm-operator oci://ghcr.io/bbd/charts/litellm-operator

# From local chart
helm install litellm-operator ./helm
```

### **🎯 Next Steps:**

1. **Create a release** to trigger the first chart publication
2. **Test the chart** locally using the Makefile targets
3. **Update your documentation** to reference the OCI installation method

The Helm chart will now automatically stay up-to-date with every release you create, making it easy for users to deploy and manage your LiteLLM operator! 🚀 