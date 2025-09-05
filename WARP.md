# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

LiteLLM Operator is a Kubernetes operator built with Kubebuilder that manages LiteLLM proxy instances and related authentication/authorization resources. It provides declarative management of:

- **LiteLLMInstance**: Core proxy deployments with database/Redis configuration
- **Model**: Individual model configurations with provider credentials
- **User**: User management with budget/rate limiting controls  
- **Team**: Team-based access control
- **VirtualKey**: API key management
- **TeamMemberAssociation**: Team membership relationships

The operator follows a controller pattern where each resource type has a dedicated reconciler that maintains desired state by making REST API calls to the LiteLLM service.

## Architecture

**Multi-Group API Structure**: 
- `api/litellm/v1alpha1/`: Core LiteLLM resources (LiteLLMInstance, Model)
- `api/auth/v1alpha1/`: Authentication resources (User, Team, VirtualKey, TeamMemberAssociation)

**Controller Pattern**:
- Controllers in `internal/controller/` implement an 8-phase reconciliation pattern:
  1. Fetch/validate resource
  2. Setup connections/clients  
  3. Handle deletion (with finalizers)
  4. Ensure finalizer exists
  5. Ensure external resource (create/update via REST API)
  6. Ensure child resources (secrets, etc.)
  7. Update status conditions
  8. Schedule periodic drift detection

**Base Controller**: All controllers inherit from `internal/controller/base/BaseController` which provides common reconciliation patterns, error handling, conditions management, and finalizer operations.

**External Integration**: Controllers use HTTP clients in `internal/litellm/` to communicate with LiteLLM REST APIs for CRUD operations on external resources.

## Development Commands

### Initial Setup
```bash
# Quick setup with Kind cluster - sets up complete dev environment
make dev-cluster-bootstrap

# Manual setup - install CRDs only
make install

# Install sample resources for testing
make install-samples        # Basic samples
make install-samples-extra  # Includes namespace and PostgreSQL
```

### Code Generation and Validation
```bash
# Core development cycle
make manifests    # Generate CRDs, RBAC, webhooks from markers
make generate     # Generate DeepCopy methods
make fmt          # Format Go code
make vet          # Static analysis
make lint         # Run golangci-lint
make lint-fix     # Auto-fix linting issues
```

### Testing
```bash
# Unit tests (excludes e2e package)
make test

# End-to-end tests (requires Kind cluster)
make test-e2e

# Run specific test packages
go test ./internal/controller/user/...
go test -run TestSpecificFunction ./path/to/package
```

### Local Development
```bash
# Build binary locally
make build

# Run controller locally (connects to configured cluster)
make run

# Build and test Docker image
make docker-build
make docker-load  # For Kind clusters
```

### Deployment
```bash
# Deploy to cluster
make deploy IMG=<registry>/litellm-operator:tag

# Generate consolidated installer
make build-installer IMG=<image>  # Creates dist/install.yaml
```

## Project Structure Patterns

**API Type Definitions**: Each resource has comprehensive spec/status definitions with kubebuilder markers for:
- CRD generation (`+kubebuilder:object:root=true`)
- Status subresource (`+kubebuilder:subresource:status`)  
- Print columns (`+kubebuilder:printcolumn`)
- Validation (`+kubebuilder:validation:`)

**Controller Reconciliation**: Controllers implement idempotent reconciliation with:
- External resource management via REST APIs
- Kubernetes secret management for credentials
- Status condition tracking (Ready, Progressing, Error)
- Finalizer-based cleanup for external resources
- Periodic drift detection (60s intervals)

**Connection References**: Resources use `ConnectionRef` pattern to reference either:
- Direct secret references (`secretRef`)
- LiteLLMInstance references (`instanceRef`)

**Naming Conventions**: `internal/util/LitellmResourceNaming` provides consistent naming for generated secrets and references.

## Testing Approach

- **Unit Tests**: Controllers have test suites using Ginkgo/Gomega in `*_test.go` files
- **Integration Tests**: `test/e2e/` contains end-to-end tests with real Kubernetes clusters
- **Test Utilities**: `test/utils/` provides common testing helpers
- **Envtest**: Uses controller-runtime's test environment for isolated Kubernetes API testing

## Development Environment

**Required Tools** (auto-downloaded to `bin/`):
- Go 1.22+
- controller-gen (CRD/RBAC generation)
- kustomize (manifest management)  
- golangci-lint (linting)
- envtest (test environment)

**Kind Integration**: Makefile provides complete Kind cluster lifecycle management for local development and testing.

**Hot Reload**: Use `make run` to run controller locally while developing - it will connect to your configured cluster context and hot-reload on changes.
