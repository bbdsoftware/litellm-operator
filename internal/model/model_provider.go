package model

import (
	"fmt"
	"strings"
)

// Enum for model providers
const (
	ModelProviderOpenAI    = "openai"
	ModelProviderAnthropic = "anthropic"
	ModelProviderGoogle    = "google"
	ModelProviderAzure     = "azure"
)

// Required fields for each provider
var (
	DefaultRequiredFields = []string{"api_key"}

	OpenAIRequiredFields    = []string{"api_key"}
	AnthropicRequiredFields = []string{"api_key"}
	GoogleRequiredFields    = []string{} // Special case: requires either api_key OR credentials
	AzureRequiredFields     = []string{"api_key", "api_base"}
)

// getRequiredFields returns the required fields for a given provider
func getRequiredFields(provider string) []string {
	switch provider {
	case ModelProviderOpenAI:
		return OpenAIRequiredFields
	case ModelProviderAnthropic:
		return AnthropicRequiredFields
	case ModelProviderGoogle:
		return GoogleRequiredFields
	case ModelProviderAzure:
		return AzureRequiredFields
	default:
		return DefaultRequiredFields
	}
}

// ModelProviderConfigValidator defines the interface for validating model provider configurations
type ModelProviderConfigValidator interface {
	ValidateConfig(config map[string]interface{}) error
	GetProviderName() string
}

// BaseValidator provides common validation functionality
type BaseValidator struct {
	providerName string
}

// NewBaseValidator creates a new base validator
func NewBaseValidator(providerName string) *BaseValidator {
	return &BaseValidator{
		providerName: providerName,
	}
}

// ValidateRequiredFields validates that all required fields are present and non-empty
func (b *BaseValidator) ValidateRequiredFields(config map[string]interface{}) error {
	requiredFields := getRequiredFields(b.providerName)

	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("required field '%s' is missing for %s provider", field, b.providerName)
		}
	}

	// Validate that required fields are not empty strings
	for _, field := range requiredFields {
		if value, exists := config[field]; exists {
			if valueStr, ok := value.(string); ok && strings.TrimSpace(valueStr) == "" {
				return fmt.Errorf("required field '%s' must be a non-empty string for %s provider", field, b.providerName)
			}
		}
	}

	return nil
}

// ValidateOptionalStringField validates that an optional string field is not empty if present
func (b *BaseValidator) ValidateOptionalStringField(config map[string]interface{}, fieldName string) error {
	if value, exists := config[fieldName]; exists {
		if valueStr, ok := value.(string); ok && strings.TrimSpace(valueStr) == "" {
			return fmt.Errorf("field '%s' cannot be empty if provided for %s provider", fieldName, b.providerName)
		}
	}
	return nil
}

// ModelProvider is a struct that contains the model provider
type ModelProvider struct {
	Provider  string
	Validator ModelProviderConfigValidator
}

type DefaultValidator struct {
	BaseValidator
}

func (v *DefaultValidator) ValidateConfig(config map[string]interface{}) error {
	return v.ValidateRequiredFields(config)
}

func (v *DefaultValidator) GetProviderName() string {
	return v.providerName
}

// NewModelProvider creates a new ModelProvider instance
func NewModelProvider(provider string) (*ModelProvider, error) {
	var validator ModelProviderConfigValidator
	switch provider {
	case ModelProviderOpenAI:
		validator = &OpenAIValidator{BaseValidator: *NewBaseValidator(ModelProviderOpenAI)}
	default:
		validator = &DefaultValidator{BaseValidator: *NewBaseValidator(provider)}
	}

	return &ModelProvider{
		Provider:  provider,
		Validator: validator,
	}, nil
}

// GetProvider returns the provider name
func (m *ModelProvider) GetProvider() string {
	return m.Provider
}

// ValidateConfig validates the configuration for the current provider
func (m *ModelProvider) ValidateConfig(config map[string]interface{}) error {
	return m.Validator.ValidateConfig(config)
}

// OpenAIValidator validates OpenAI provider configurations
type OpenAIValidator struct {
	BaseValidator
}

func (v *OpenAIValidator) GetProviderName() string {
	return v.providerName
}

func (v *OpenAIValidator) ValidateConfig(config map[string]interface{}) error {
	// Validate required fields
	if err := v.ValidateRequiredFields(config); err != nil {
		return err
	}

	// Validate optional fields
	if err := v.ValidateOptionalStringField(config, "api_base"); err != nil {
		return err
	}

	return nil
}
