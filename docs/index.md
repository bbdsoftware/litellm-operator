# LiteLLM Operator

A Kubernetes operator for managing [LiteLLM](https://github.com/BerriAI/litellm) resources in your cluster.

## Overview

The LiteLLM Operator simplifies the management of LiteLLM resources in Kubernetes environments by providing custom resource definitions (CRDs) and controllers for:

- **Virtual Keys** - API key management for LiteLLM proxy
- **Users** - User account management and authentication
- **Teams** - Team-based access control and organization
- **Team Member Associations** - Relationships between users and teams

## Architecture

The operator is designed to work alongside your LiteLLM deployment, providing a Kubernetes-native way to manage authentication and authorization resources.

![Architecture Overview](assets/lite-llm-architecture2.png)

## Key Features

- 🔐 **Secure API Key Management** - Create and manage virtual keys for LiteLLM proxy access
- 👥 **User Management** - Define and manage user accounts with role-based access
- 🏢 **Team Organization** - Group users into teams with shared permissions
- 🔗 **Association Management** - Flexible user-team relationships
- 🚀 **Kubernetes Native** - Built using the Operator SDK with standard Kubernetes practices
- 📊 **Observability** - Built-in metrics and monitoring capabilities

## Quick Start

Ready to get started? Check out our [installation guide](getting-started/installation.md) to deploy the operator in your cluster.

```bash
# Install CRDs
make install

# Deploy the operator
make deploy IMG=<your-registry>/litellm-operator:latest
```

## Resources

- [Getting Started](getting-started/installation.md) - Installation and setup
- [User Guide](user-guide/virtual-keys.md) - How to use the operator resources
- [Developer Guide](developer-guide/architecture.md) - Architecture and development info
- [Release Process](developer-guide/release-process.md) - How to create and manage releases
- [API Reference](reference/api.md) - Complete API documentation

## Community

- 🐛 [Report Issues](https://github.com/yourusername/litellm-operator/issues)
- 💬 [Discussions](https://github.com/yourusername/litellm-operator/discussions)
- 🤝 [Contributing](developer-guide/contributing.md)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](https://github.com/litellm-io/litellm-operator/blob/main/LICENSE) file for details. 