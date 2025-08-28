package model

import (
	"testing"
)

func TestModelProvider_GetValidator(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{
			name:     "OpenAI provider",
			provider: ModelProviderOpenAI,
			wantErr:  false,
		},
		{
			name:     "Anthropic provider",
			provider: ModelProviderAnthropic,
			wantErr:  false,
		},
		{
			name:     "Google provider",
			provider: ModelProviderGoogle,
			wantErr:  false,
		},
		{
			name:     "Azure provider",
			provider: ModelProviderAzure,
			wantErr:  false,
		},
		{
			name:     "Unsupported provider",
			provider: "unsupported",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := NewModelProvider(tt.provider)
			validator, err := mp.GetValidator()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for provider %s, but got none", tt.provider)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for provider %s: %v", tt.provider, err)
				return
			}

			if validator == nil {
				t.Errorf("Expected validator for provider %s, but got nil", tt.provider)
				return
			}

			if validator.GetProviderName() != tt.provider {
				t.Errorf("Expected provider name %s, but got %s", tt.provider, validator.GetProviderName())
			}
		})
	}
}

func TestOpenAIValidator_ValidateConfig(t *testing.T) {
	validator := &OpenAIValidator{}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid OpenAI config",
			config: map[string]interface{}{
				"api_key": "sk-1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name: "Valid OpenAI config with optional fields",
			config: map[string]interface{}{
				"api_key":  "sk-1234567890abcdef",
				"api_base": "https://api.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "Missing API key",
			config: map[string]interface{}{
				"api_base": "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "Empty API key",
			config: map[string]interface{}{
				"api_key": "",
			},
			wantErr: true,
		},
		{
			name: "Empty API base URL",
			config: map[string]interface{}{
				"api_key":  "sk-1234567890abcdef",
				"api_base": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenAIValidator.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnthropicValidator_ValidateConfig(t *testing.T) {
	validator := &AnthropicValidator{}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid Anthropic config",
			config: map[string]interface{}{
				"api_key": "sk-ant-1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name: "Valid Anthropic config with optional fields",
			config: map[string]interface{}{
				"api_key":  "sk-ant-1234567890abcdef",
				"api_base": "https://api.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "Missing API key",
			config: map[string]interface{}{
				"api_base": "https://api.anthropic.com",
			},
			wantErr: true,
		},
		{
			name: "Empty API key",
			config: map[string]interface{}{
				"api_key": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnthropicValidator.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoogleValidator_ValidateConfig(t *testing.T) {
	validator := &GoogleValidator{}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid Google config with API key",
			config: map[string]interface{}{
				"api_key": "AIzaSyC1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name: "Valid Google config with credentials",
			config: map[string]interface{}{
				"credentials": "service-account-key.json",
			},
			wantErr: false,
		},
		{
			name: "Valid Google config with all fields",
			config: map[string]interface{}{
				"api_key":  "AIzaSyC1234567890abcdef",
				"project":  "my-project",
				"location": "us-central1",
			},
			wantErr: false,
		},
		{
			name: "Missing both API key and credentials",
			config: map[string]interface{}{
				"project": "my-project",
			},
			wantErr: true,
		},
		{
			name: "Empty project ID",
			config: map[string]interface{}{
				"api_key": "AIzaSyC1234567890abcdef",
				"project": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("GoogleValidator.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAzureValidator_ValidateConfig(t *testing.T) {
	validator := &AzureValidator{}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid Azure config",
			config: map[string]interface{}{
				"api_key":  "sk-1234567890abcdef",
				"api_base": "https://my-resource.openai.azure.com",
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with API version",
			config: map[string]interface{}{
				"api_key":     "sk-1234567890abcdef",
				"api_base":    "https://my-resource.openai.azure.com",
				"api_version": "2023-05-15",
			},
			wantErr: false,
		},
		{
			name: "Missing API key",
			config: map[string]interface{}{
				"api_base": "https://my-resource.openai.azure.com",
			},
			wantErr: true,
		},
		{
			name: "Missing API base",
			config: map[string]interface{}{
				"api_key": "sk-1234567890abcdef",
			},
			wantErr: true,
		},
		{
			name: "Empty API key",
			config: map[string]interface{}{
				"api_key":  "",
				"api_base": "https://my-resource.openai.azure.com",
			},
			wantErr: true,
		},
		{
			name: "Empty API base",
			config: map[string]interface{}{
				"api_key":  "sk-1234567890abcdef",
				"api_base": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("AzureValidator.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		config   map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "Valid OpenAI config",
			provider: ModelProviderOpenAI,
			config: map[string]interface{}{
				"api_key": "sk-1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name:     "Valid Anthropic config",
			provider: ModelProviderAnthropic,
			config: map[string]interface{}{
				"api_key": "sk-ant-1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name:     "Valid Google config",
			provider: ModelProviderGoogle,
			config: map[string]interface{}{
				"api_key": "AIzaSyC1234567890abcdef",
			},
			wantErr: false,
		},
		{
			name:     "Valid Azure config",
			provider: ModelProviderAzure,
			config: map[string]interface{}{
				"api_key":  "sk-1234567890abcdef",
				"api_base": "https://my-resource.openai.azure.com",
			},
			wantErr: false,
		},
		{
			name:     "Invalid OpenAI config",
			provider: ModelProviderOpenAI,
			config: map[string]interface{}{
				"api_base": "https://api.openai.com/v1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := NewModelProvider(tt.provider)
			err := mp.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelProvider.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
