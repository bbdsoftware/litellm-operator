## Getting Started

It is expected that the operator will be deployed in the same namespace as the litellm service.

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- Kind (optional, for local development)

### Local Development Setup

#### Quick Start with Kind Cluster
For local development, you can use the provided Makefile targets to set up a complete development environment:

```sh
# Create a Kind cluster and bootstrap the development environment
make dev-cluster-bootstrap

# Or recreate the entire cluster if needed
make dev-cluster-recreate
```

This will:
1. Create a Kind cluster (if it doesn't exist)
2. Generate manifests and install CRDs
3. Install extra samples and basic samples
4. Set up the complete development environment
5. You will be able to ruin the controller locally thrp you IDE of choise

#### Manual Development Setup
If you prefer to set up manually or use an existing cluster:

**Generate manifests and install CRDs:**
```sh
make install
```

**Install sample resources:**
```sh
# Install basic samples
make install-samples

# Install extra samples (includes namespace and PostgreSQL cluster)
make install-samples-extra
```

### Development Workflow

#### Code Generation and Validation
```sh
# Generate manifests (CRDs, RBAC, webhooks)
make manifests

# Generate code (DeepCopy methods)
make generate

# Format code
make fmt

# Run static analysis
make vet

# Run linter
make lint

# Run linter with auto-fixes
make lint-fix
```

#### Testing
```sh
# Run unit tests
make test

# Run end-to-end tests (requires Kind cluster)
make test-e2e
```

#### Building
```sh
# Build the manager binary locally
make build

# Run the controller locally (for development) in terminal
make run

# Build Docker image
make docker-build

# Build for multiple platforms
make docker-buildx
```

### Deployment

#### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/litellm-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/litellm-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
# Install basic samples
kubectl apply -k config/samples/

# Or use the Makefile target
make install-samples
```

>**NOTE**: Ensure that the samples has default values to test it out.

#### Alternative Deployment Methods

**Build and load image to Kind cluster:**
```sh
make docker-build docker-load IMG=controller:latest
```

**Generate consolidated installer:**
```sh
make build-installer IMG=<your-image>
# This creates dist/install.yaml with CRDs and deployment
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
# Remove samples
kubectl delete -k config/samples/

# Or use the Makefile target
make uninstall-samples
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

**Clean up Kind cluster (if using local development):**
```sh
make kind-cluster-delete
```

### Development Tools

The project includes several development tools that are automatically managed:

- **controller-gen**: Generates CRDs, RBAC, and webhook configurations
- **kustomize**: Manages Kubernetes manifests
- **envtest**: Provides test environment for controller-runtime
- **golangci-lint**: Code linting and analysis
- **crd-ref-docs**: Generates CRD documentation
- **openapi-generator**: Generates OpenAPI specifications

These tools are automatically downloaded to the `bin/` directory when needed.

### Environment Variables

Key environment variables for development:

- `IMG`: Docker image for the controller (default: `controller:latest`)
- `VERSION`: Project version for bundles (default: `0.0.1`)
- `CONTAINER_TOOL`: Container tool to use (default: `docker`)
- `LOCALBIN`: Directory for development tools (default: `./bin`)

### Help

To see all available Makefile targets and their descriptions:

```sh
make help
```