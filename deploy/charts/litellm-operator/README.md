# LiteLLM Operator

A Helm chart for deploying the LiteLLM Operator, a Kubernetes operator for managing LiteLLM resources including virtual keys, users, teams, and model configurations.

## Values

| Key | Type | Default | Description |
|-----------|------|---------|-------------|
| `controllerManager.manager.image.repository` | string | `ghcr.io/bbdsoftware/litellm-operator` | Controller image repository |
| `controllerManager.manager.resources.limits.cpu` | string | `200m` | CPU resource limit |
| `controllerManager.manager.resources.limits.memory` | string | `128Mi` | Memory resource limit |
| `controllerManager.manager.resources.requests.cpu` | string | `50m` | CPU resource request |
| `controllerManager.manager.resources.requests.memory` | string | `64Mi` | Memory resource request |
| `controllerManager.replicas` | int | `1` | Number of controller replicas |
| `controllerManager.podSecurityContext.runAsNonRoot` | boolean | `true` | Run as non-root user |
| `kubernetesClusterDomain` | string | `cluster.local` | Kubernetes cluster domain |
| `metricsService.ports[0].port` | int | `8443` | Metrics service port |
| `metricsService.type` | string | `ClusterIP` | Metrics service type |

## Custom Resource Definitions (CRDs)

The chart installs the following CRDs:

- **LiteLLMInstance** (`litellminstances.litellm.litellm.ai`) - Manages LiteLLM service instances
- **VirtualKey** (`virtualkeys.auth.litellm.ai`) - Manages virtual API keys
- **User** (`users.auth.litellm.ai`) - Manages user accounts
- **Team** (`teams.auth.litellm.ai`) - Manages team configurations
- **TeamMemberAssociation** (`teammemberassociations.auth.litellm.ai`) - Manages team memberships
- **Model** (`models.litellm.litellm.ai`) - Manages model configurations
