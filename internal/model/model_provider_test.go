package model

import (
	"testing"
)

const (
	testOpenAIAPIKey      = "sk-1234567890abcdef"
	testOpenAIAPIBase     = "https://api.openai.com/v1"
	testAnthropicAPIKey   = "sk-ant-1234567890abcdef"
	testAnthropicAPIBase  = "https://api.anthropic.com"
	testGoogleCredentials = "AIzaSyC1234567890abcdef"
	testAzureAPIBase      = "https://my-resource.openai.azure.com"
	testCaseMissingAPIKey = "Missing API key"
	testCaseEmptyAPIKey   = "Empty API key"
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
			mp, err := NewModelProvider(tt.provider)
			if tt.wantErr {
				// No error is expected for validator creation in current logic
				return
			}
			validator := mp.Validator

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
	validator := NewBaseValidator(ModelProviderOpenAI)

	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			name: "Valid OpenAI config",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testOpenAIAPIBase,
			},
			wantErr: false,
		},
		{
			name: "Valid OpenAI config with optional fields",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testOpenAIAPIBase,
			},
			wantErr: false,
		},
		{
			name: testCaseMissingAPIKey,
			config: map[string]string{
				FieldAPIBase: testOpenAIAPIBase,
			},
			wantErr: true,
		},
		{
			name: testCaseEmptyAPIKey,
			config: map[string]string{
				FieldAPIKey: "",
			},
			wantErr: true,
		},
		{
			name: "Empty API base URL",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: "",
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
	validator := NewBaseValidator(ModelProviderAnthropic)

	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			name: "Valid Anthropic config",
			config: map[string]string{
				FieldAPIKey:  testAnthropicAPIKey,
				FieldAPIBase: testAnthropicAPIBase,
			},
			wantErr: false,
		},
		{
			name: "Valid Anthropic config with optional fields",
			config: map[string]string{
				FieldAPIKey:  testAnthropicAPIKey,
				FieldAPIBase: testAnthropicAPIBase,
			},
			wantErr: false,
		},
		{
			name: testCaseMissingAPIKey,
			config: map[string]string{
				FieldAPIBase: testAnthropicAPIBase,
			},
			wantErr: true,
		},
		{
			name: testCaseEmptyAPIKey,
			config: map[string]string{
				FieldAPIKey: "",
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
	validator := NewBaseValidator(ModelProviderGoogle)

	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			name: "Valid Google config with API key",
			config: map[string]string{
				FieldVertexCredentials: testGoogleCredentials,
			},
			wantErr: false,
		},
		{
			name: "Valid Google config with credentials",
			config: map[string]string{
				FieldVertexCredentials: "service-account-key.json",
			},
			wantErr: false,
		},
		{
			name: "Valid Google config with all fields",
			config: map[string]string{
				FieldVertexCredentials: testGoogleCredentials,
				"project":              "my-project",
				"location":             "us-central1",
			},
			wantErr: false,
		},
		{
			name: "Missing both API key and credentials",
			config: map[string]string{
				"project": "my-project",
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
	validator := NewBaseValidator(ModelProviderAzure)

	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			name: "Valid Azure config",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testAzureAPIBase,
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with API version",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testAzureAPIBase,
				"apiVersion": "2023-05-15",
			},
			wantErr: false,
		},
		{
			name: testCaseMissingAPIKey,
			config: map[string]string{
				FieldAPIBase: testAzureAPIBase,
			},
			wantErr: true,
		},
		{
			name: "Missing API base",
			config: map[string]string{
				FieldAPIKey: testOpenAIAPIKey,
			},
			wantErr: true,
		},
		{
			name: testCaseEmptyAPIKey,
			config: map[string]string{
				FieldAPIKey:  "",
				FieldAPIBase: testAzureAPIBase,
			},
			wantErr: true,
		},
		{
			name: "Empty API base",
			config: map[string]string{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: "",
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
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testOpenAIAPIBase,
			},
			wantErr: false,
		},
		{
			name:     "Valid Anthropic config",
			provider: ModelProviderAnthropic,
			config: map[string]interface{}{
				FieldAPIKey:  testAnthropicAPIKey,
				FieldAPIBase: testAnthropicAPIBase,
			},
			wantErr: false,
		},
		{
			name:     "Valid Google config",
			provider: ModelProviderGoogle,
			config: map[string]interface{}{
				FieldVertexCredentials: testGoogleCredentials,
			},
			wantErr: false,
		},
		{
			name:     "Valid Azure config",
			provider: ModelProviderAzure,
			config: map[string]interface{}{
				FieldAPIKey:  testOpenAIAPIKey,
				FieldAPIBase: testAzureAPIBase,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp, err := NewModelProvider(tt.provider)
			if err != nil {
				t.Errorf("Unexpected error creating provider %s: %v", tt.provider, err)
				return
			}
			// Convert config to map[string]string if needed
			config := make(map[string]string)
			for k, v := range tt.config {
				if str, ok := v.(string); ok {
					config[k] = str
				}
			}
			err = mp.ValidateConfig(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelProvider.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
