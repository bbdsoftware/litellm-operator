# Virtual Keys

Virtual Keys are API keys that provide controlled access to your LiteLLM proxy. They enable fine-grained access control with budget limits, model restrictions, and usage monitoring.

## Overview

Virtual Keys in the LiteLLM Operator provide:

- **Budget Control** - Set spending limits per key
- **Model Access** - Restrict which models can be used
- **User Association** - Link keys to specific users
- **Time Limits** - Set expiration dates
- **Model Aliases** - Map public model names to your internal deployments

## Creating Virtual Keys

### Basic Virtual Key

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: basic-key
  namespace: default
spec:
  userId: "user@example.com"
  maxBudget: 100.0
  duration: "30d"
```

### Advanced Virtual Key

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: advanced-key
  namespace: default
spec:
  userId: "user@example.com"
  maxBudget: 250.0
  models:
    - "gpt-3.5-turbo"
    - "gpt-4"
    - "claude-3-sonnet"
  aliases:
    gpt-3.5-turbo: "azure/gpt-35-turbo-16k"
    gpt-4: "azure/gpt-4-32k"
    claude-3-sonnet: "bedrock/claude-3-sonnet"
  duration: "90d"
  tpmLimit: 10000
  rpmLimit: 100
  metadata:
    team: "ai-research"
    department: "engineering"
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `userId` | string | User ID this key belongs to | Yes |
| `maxBudget` | float | Maximum spend limit in dollars | Yes |
| `models` | []string | Allowed models (empty = all models) | No |
| `aliases` | map[string]string | Model name mappings | No |
| `duration` | string | Key lifetime (e.g., "30d", "6h") | No |
| `tpmLimit` | int | Tokens per minute limit | No |
| `rpmLimit` | int | Requests per minute limit | No |
| `metadata` | map[string]string | Custom metadata | No |

## Managing Virtual Keys

### List Virtual Keys

```bash
kubectl get virtualkeys
```

### Get Key Details

```bash
kubectl describe virtualkey my-key
```

### Get Key Value

```bash
kubectl get virtualkey my-key -o jsonpath='{.status.keyValue}'
```

### Update a Virtual Key

```bash
kubectl patch virtualkey my-key --type='merge' -p='{"spec":{"maxBudget":200.0}}'
```

### Delete a Virtual Key

```bash
kubectl delete virtualkey my-key
```

## Usage Examples

### Using the Virtual Key

Once created, use the virtual key to authenticate with LiteLLM:

```bash
# Get the key value
KEY=$(kubectl get virtualkey my-key -o jsonpath='{.status.keyValue}')

# Make API call
curl -X POST "https://your-litellm-proxy.com/chat/completions" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'
```

### Python Example

```python
import openai

# Configure client with virtual key
client = openai.OpenAI(
    api_key="<your-virtual-key>",
    base_url="https://your-litellm-proxy.com"
)

# Make API call
response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

## Monitoring and Troubleshooting

### Check Key Status

```bash
kubectl get virtualkey my-key -o yaml
```

Look for status fields:
- `status.keyValue` - The actual API key
- `status.currentSpend` - Current usage
- `status.isActive` - Whether key is active
- `status.expiryDate` - When key expires

### Common Issues

**Key Not Working**
- Check if key has expired
- Verify budget hasn't been exceeded
- Ensure user exists and is active

**Budget Exceeded**
- Check current spend: `kubectl get virtualkey my-key -o jsonpath='{.status.currentSpend}'`
- Increase budget if needed
- Monitor usage patterns

**Model Access Denied**
- Verify model is in allowed list
- Check model aliases are correct
- Ensure LiteLLM proxy has access to the model

## Best Practices

### Security
- Rotate keys regularly
- Use appropriate budget limits
- Monitor key usage
- Store keys securely in your applications

### Cost Control
- Set realistic budget limits
- Use model restrictions to prevent expensive model usage
- Monitor spend across all keys
- Set up alerts for budget thresholds

### Organization
- Use consistent naming conventions
- Add metadata for tracking
- Associate keys with users/teams
- Document key purposes

## Next Steps

- Learn about [Users](users.md) and [Teams](teams.md)
- Set up [monitoring and alerts](../reference/monitoring.md)
- Review [security best practices](../community/security.md) 