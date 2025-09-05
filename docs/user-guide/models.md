# Models

Models represent LLM configurations in your LiteLLM system. They provide the foundation for routing requests to different model providers and managing model-specific configurations.

## Overview

Model resources in the LiteLLM Operator provide:

- **Model Configuration** - Define model routing and settings
- **Provider Management** - Configure model provider credentials and endpoints 
- **Request Routing** - Control how requests are routed to models
 # Models

Models represent LLM configurations in your LiteLLM system. They provide the foundation for routing requests to different model providers and managing model-specific configurations.

## Overview

Model resources in the LiteLLM Operator provide:

- **Model Configuration** - Define model routing and settings
- **Provider Management** - Configure model provider credentials and endpoints
- **Request Routing** - Control how requests are routed to models
- **Usage Tracking** - Monitor model usage and performance
- **Auto-Routing** - Configure automatic routing between models

## Creating Models

### Basic Model

 # Models

Models represent LLM configurations in your LiteLLM system. They provide the foundation for routing requests to different model providers and managing model-specific configurations.

## Overview

Model resources in the LiteLLM Operator provide:

- **Model Configuration** - Define model routing and settings
- **Provider Management** - Configure model provider credentials and endpoints
- **Request Routing** - Control how requests are routed to models
- **Usage Tracking** - Monitor model usage and performance
- **Auto-Routing** - Configure automatic routing between models

## Creating Models

### Basic Model

```yaml
apiVersion: litellm.litellm.ai/v1alpha1
kind: Model
metadata:
  name: gpt4-model
spec:
  modelName: "gpt-4"
  modelSecretRef:
    secretName: openai-secret
    namespace: litellm
  litellmParams:
    model: "gpt-4"
    timeout: 30
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

Sensitive data like API keys, API URLs, AWS secrets, etc. should be stored in the `modelSecretRef`.


## Specification Reference

Note: `connectionRef` is required — provide either `connectionRef.secretRef` (to point at provider credentials) or `connectionRef.instanceRef` (to point at a LiteLLM instance). The choice is up to you; at least one must be present.

Also note: `litellmParams.model` is required for each Model and should contain the provider's model identifier (for example: "gpt-4").
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `connectionRef` | object | Reference to connection info (secretRef and instanceRef). Must include either `secretRef` or `instanceRef`. | Yes |
| `connectionRef.secretRef` | object | Secret reference that contains provider credentials. | No |
| `connectionRef.secretRef.namespace` | string | Namespace of the secret. | No |
| `connectionRef.secretRef.secretName` | string | Name of the secret. | No |
| `connectionRef.instanceRef` | object | Reference to a LiteLLM instance resource. | No |
| `connectionRef.instanceRef.namespace` | string | Namespace of the LiteLLM instance. | No |
| `connectionRef.instanceRef.name` | string | Name of the LiteLLM instance. | No |
| `modelName` | string | Human-friendly name identifying the Model resource. | No |
| `modelSecretRef` | object | Secret reference that provides provider-specific credentials for this model. | Yes (present in spec) |
| `modelSecretRef.namespace` | string | Namespace of the model secret. | No |
| `modelSecretRef.secretName` | string | Name of the model secret. | No |
| `litellmParams.inputCostPerToken` | string | Cost per input token. | No |
| `litellmParams.outputCostPerToken` | string | Cost per output token. | No |
| `litellmParams.inputCostPerSecond` | string | Cost per second for input. | No |
| `litellmParams.outputCostPerSecond` | string | Cost per second for output. | No |
| `litellmParams.inputCostPerPixel` | string | Cost per input pixel (for vision models). | No |
| `litellmParams.outputCostPerPixel` | string | Cost per output pixel (for vision models). | No |
| `litellmParams.apiKey` | string | API key for the model provider. | No |
| `litellmParams.apiBase` | string | Base URL for the provider API. | No |
| `litellmParams.apiVersion` | string | API version to use. | No |
| `litellmParams.vertexProject` | string | Google Cloud project for Vertex AI. | No |
| `litellmParams.vertexLocation` | string | GCP location for Vertex AI. | No |
| `litellmParams.vertexCredentials` | string | Credentials for Vertex AI (encoded or reference). | No |
| `litellmParams.regionName` | string | Generic region name for provider services. | No |
| `litellmParams.awsAccessKeyId` | string | AWS access key ID for AWS-hosted providers. | No |
| `litellmParams.awsSecretAccessKey` | string | AWS secret access key. | No |
| `litellmParams.awsRegionName` | string | AWS region name. | No |
| `litellmParams.watsonxRegionName` | string | Region name for WatsonX deployments. | No |
| `litellmParams.customLLMProvider` | string | Identifier for a custom LLM provider. Use this if your model is not hosted by the provider in your `litellmParams.model` | No |
| `litellmParams.tpm` | int | Tokens per minute rate limit. | No |
| `litellmParams.rpm` | int | Requests per minute rate limit. | No |
| `litellmParams.timeout` | int | Request timeout in seconds. | No |
| `litellmParams.streamTimeout` | int | Stream timeout in seconds for streaming responses. | No |
| `litellmParams.maxRetries` | int | Maximum number of retry attempts for requests. | No |
| `litellmParams.organization` | string | Organization identifier (provider-specific). | No |
| `litellmParams.configurableClientsideAuthParams` | array/object | Client-side auth parameters as raw extensions (configurable). | No |
| `litellmParams.litellmCredentialName` | string | Name of a LiteLLM-stored credential to use. | No |
| `litellmParams.litellmTraceId` | string | Trace ID to attach to requests for correlation. | No |
| `litellmParams.maxFileSizeMb` | int | Maximum allowed file size in MB for uploads. | No |
| `litellmParams.maxBudget` | string | Maximum budget for model usage (string to allow units). | No |
| `litellmParams.budgetDuration` | string | Time window for the budget (e.g., "1h", "30m"). | No |
| `litellmParams.useInPassThrough` | bool | Whether to allow pass-through usage of this model. | No |
| `litellmParams.useLitellmProxy` | bool | Whether to route requests through LiteLLM proxy. | No |
| `litellmParams.mergeReasoningContentInChoices` | bool | Merge reasoning/content into choice outputs when present. | No |
| `litellmParams.mockResponse` | string | Mock response to use for testing. | No |
| `litellmParams.autoRouterConfigPath` | string | Path to auto-router configuration file. | No |
| `litellmParams.autoRouterConfig` | string | Inline auto-router configuration JSON/string. | No |
| `litellmParams.autoRouterDefaultModel` | string | Default model to use for auto-routing decisions. | No |
| `litellmParams.autoRouterEmbeddingModel` | string | Embedding model used by auto-router. | No |
| `litellmParams.model` | string | Underlying provider model identifier. Naming convention should be <provider>/<base-model> (e.g., "openai/gpt-4"). | Yes |
| `modelInfo.id` | string | Server-provided model UUID (returned in status). | No |
| `modelInfo.dbModel` | bool | Whether this model is stored as a DB model. | No |
| `modelInfo.teamId` | string | Team identifier associated with the model. | No |
| `modelInfo.teamPublicModelName` | string | Team-visible public model name. | No |

### Managing Models
Listing Models
```
kubectl get models
```
Describing a Model
```
kubectl describe model gpt-model
```
Deleting a Model
```
kubectl delete model gpt-model
```

### Next Steps
- Learn about [Virtual Keys](virtual-keys.md) for model access
- Configure [Users](users.md) and [Teams](teams.md)
- Review [security best practices](../community/security.md)

