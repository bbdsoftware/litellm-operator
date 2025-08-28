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
  litellmParams:
    model: "gpt-4"
    api_base: "https://api.openai.com/v1"
    timeout: 30
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

### Specification Reference
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `connectionRef` | object | Reference to connection info (secretRef and instanceRef). Use to point the Model to a LiteLLM instance or secret. | No |
| `connectionRef.secretRef` | object | Secret reference that contains provider credentials. | No |
| `connectionRef.secretRef.namespace` | string | Namespace of the secret. | No |
| `connectionRef.secretRef.secretName` | string | Name of the secret. | No |
| `connectionRef.instanceRef` | object | Reference to a LiteLLM instance resource. | No |
| `connectionRef.instanceRef.namespace` | string | Namespace of the LiteLLM instance. | No |
| `connectionRef.instanceRef.name` | string | Name of the LiteLLM instance. | No |
| `model_name` | string | Human-friendly name identifying the Model resource. | No |
| `litellm_params.input_cost_per_token` | string | Cost per input token. | No |
| `litellm_params.output_cost_per_token` | string | Cost per output token. | No |
| `litellm_params.input_cost_per_second` | string | Cost per second for input. | No |
| `litellm_params.output_cost_per_second` | string | Cost per second for output. | No |
| `litellm_params.input_cost_per_pixel` | string | Cost per input pixel (for vision models). | No |
| `litellm_params.output_cost_per_pixel` | string | Cost per output pixel (for vision models). | No |
| `litellm_params.api_key` | string | API key for the model provider. | No |
| `litellm_params.api_base` | string | Base URL for the provider API. | No |
| `litellm_params.api_version` | string | API version to use. | No |
| `litellm_params.vertex_project` | string | Google Cloud project for Vertex AI. | No |
| `litellm_params.vertex_location` | string | GCP location for Vertex AI. | No |
| `litellm_params.vertex_credentials` | string | Credentials for Vertex AI (encoded or reference). | No |
| `litellm_params.region_name` | string | Generic region name for provider services. | No |
| `litellm_params.aws_access_key_id` | string | AWS access key ID for AWS-hosted providers. | No |
| `litellm_params.aws_secret_access_key` | string | AWS secret access key. | No |
| `litellm_params.aws_region_name` | string | AWS region name. | No |
| `litellm_params.watsonx_region_name` | string | Region name for WatsonX deployments. | No |
| `litellm_params.custom_llm_provider` | string | Identifier for a custom LLM provider. | No |
| `litellm_params.tpm` | int | Tokens per minute rate limit. | No |
| `litellm_params.rpm` | int | Requests per minute rate limit. | No |
| `litellm_params.timeout` | int | Request timeout in seconds. | No |
| `litellm_params.stream_timeout` | int | Stream timeout in seconds for streaming responses. | No |
| `litellm_params.max_retries` | int | Maximum number of retry attempts for requests. | No |
| `litellm_params.organization` | string | Organization identifier (provider-specific). | No |
| `litellm_params.configurable_clientside_auth_params` | array/object | Client-side auth parameters as raw extensions (configurable). | No |
| `litellm_params.litellm_credential_name` | string | Name of a LiteLLM-stored credential to use. | No |
| `litellm_params.litellm_trace_id` | string | Trace ID to attach to requests for correlation. | No |
| `litellm_params.max_file_size_mb` | int | Maximum allowed file size in MB for uploads. | No |
| `litellm_params.max_budget` | string | Maximum budget for model usage (string to allow units). | No |
| `litellm_params.budget_duration` | string | Time window for the budget (e.g., "1h", "30m"). | No |
| `litellm_params.use_in_pass_through` | bool | Whether to allow pass-through usage of this model. | No |
| `litellm_params.use_litellm_proxy` | bool | Whether to route requests through LiteLLM proxy. | No |
| `litellm_params.merge_reasoning_content_in_choices` | bool | Merge reasoning/content into choice outputs when present. | No |
| `litellm_params.model_info` | object/raw | Additional model metadata (raw extension). | No |
| `litellm_params.mock_response` | string | Mock response to use for testing. | No |
| `litellm_params.auto_router_config_path` | string | Path to auto-router configuration file. | No |
| `litellm_params.auto_router_config` | string | Inline auto-router configuration JSON/string. | No |
| `litellm_params.auto_router_default_model` | string | Default model to use for auto-routing decisions. | No |
| `litellm_params.auto_router_embedding_model` | string | Embedding model used by auto-router. | No |
| `litellm_params.model` | string | Underlying provider model identifier (e.g., "gpt-4"). | No |
| `model_info.id` | string | Server-provided model UUID. | No |
| `model_info.db_model` | bool | Whether this model is stored as a DB model. | No |
| `model_info.team_id` | string | Team identifier associated with the model. | No |
| `model_info.team_public_model_name` | string | Team-visible public model name. | No |

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

