# Virtual Keys

Virtual Keys are API keys that provide controlled access to your LiteLLM proxy. They enable fine-grained access control with budget limits, model restrictions, and usage monitoring.

## Overview

Virtual Keys in the LiteLLM Operator provide:

- **Budget Control** - Set spending limits per key
- **Model Access** - Restrict which models can be used
- **Time Limits** - Set expiration dates and budget durations
- **Key Management** - Organise keys with aliases for easy identification

## Creating Virtual Keys

### Basic Virtual Key

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: example-service
spec:
  keyAlias: example-service
  models:
    - gpt-4o
  maxBudget: "10"
  budgetDuration: 1h
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

### Advanced Virtual Key

```yaml
apiVersion: auth.litellm.ai/v1alpha1
kind: VirtualKey
metadata:
  name: research-key
spec:
  keyAlias: research-key
  models:
    - gpt-4o
    - claude-3-sonnet
    - gemini-pro
  maxBudget: "250"
  budgetDuration: 30d
  connectionRef:
    instanceRef:
      name: litellm-example
      namespace: litellm
```

## Specification Reference

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `keyAlias` | string | Unique identifier for the key | Yes |
| `models` | []string | Allowed models (empty = all models) | No |
| `maxBudget` | string | Maximum spend limit in dollars | Yes |
| `budgetDuration` | string | Budget duration (e.g., "1h", "30d") | Yes |
| `connectionRef` | object | Reference to LiteLLM instance | Yes |

## Managing Virtual Keys

### List Virtual Keys

```bash
kubectl get virtualkeys
```

### Get Key Details

```bash
kubectl describe virtualkey example-service
```

### Get Key Value

```bash
KEY_SECRET=$(kubectl get virtualkey example-service -o jsonpath='{.status.keySecretRef}')

KEY=$(kubectl get secret $KEY_SECRET -o jsonpath='{.data.key}' | base64 -d)
```

### Update a Virtual Key

```bash
kubectl patch virtualkey example-service --type='merge' -p='{"spec":{"maxBudget":"200"}}'
```

### Delete a Virtual Key

```bash
kubectl delete virtualkey example-service
```

## Usage Examples

### Using the Virtual Key

Once created, use the virtual key to authenticate with LiteLLM:

```bash
# Get the key value
KEY_SECRET=$(kubectl get virtualkey example-service -o jsonpath='{.status.keySecretRef}')

KEY=$(kubectl get secret $KEY_SECRET -o jsonpath='{.data.key}' | base64 -d)

# Make API call
curl -X POST "https://your-litellm-proxy.com/chat/completions" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
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
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

## Monitoring and Troubleshooting

### Check Key Status

```bash
kubectl get virtualkey example-service -o yaml
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
- Ensure LiteLLM instance is running and accessible

**Budget Exceeded**
- Check current spend: `kubectl get virtualkey example-service -o jsonpath='{.status.currentSpend}'`
- Increase budget if needed
- Monitor usage patterns

**Model Access Denied**
- Verify model is in allowed list
- Check that the LiteLLM instance has access to the model
- Ensure model name matches exactly

## Best Practices

### Security
- Use descriptive key aliases for easy identification
- Set appropriate budget limits and durations
- Monitor key usage regularly
- Store keys securely in your applications

### Cost Control
- Set realistic budget limits based on expected usage
- Use model restrictions to prevent expensive model usage
- Monitor spend across all keys
- Set up alerts for budget thresholds

### Organisation
- Use consistent naming conventions for key aliases
- Group keys by purpose or team
- Document key purposes and usage patterns
- Regularly review and clean up unused keys

## Integration with Users and Teams

Virtual Keys can be created automatically when users are created (using `autoCreateKey: true` in User resources) or created independently for service accounts and applications.

### Auto-Created Keys

When a User resource has `autoCreateKey: true`, a Virtual Key is automatically created with:
- `keyAlias` matching the user's `keyAlias`
- Same model access as the user
- Same budget and duration settings
- Same LiteLLM instance connection

## Next Steps

- Learn about [Users](users.md) and [Teams](teams.md)
- Review [security best practices](../community/security.md) 