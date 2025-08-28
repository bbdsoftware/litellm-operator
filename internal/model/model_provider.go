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
	ModelProviderAWS       = "aws"
	ModelProviderBedrock   = "bedrock"
)

// Required fields for each provider
var (
	DefaultRequiredFields = []string{"apiKey", "apiBase"}
	GoogleRequiredFields  = []string{"vertexCredentials"}
	AwsRequiredFields     = []string{"awsSecretAccessKey", "awsAccessKeyId"}
)

var providerRequiredFields = map[string][]string{
	ModelProviderOpenAI:    DefaultRequiredFields,
	ModelProviderAnthropic: DefaultRequiredFields,
	ModelProviderAzure:     DefaultRequiredFields,
	ModelProviderGoogle:    GoogleRequiredFields,
	ModelProviderAWS:       AwsRequiredFields,
	ModelProviderBedrock:   AwsRequiredFields,
}

// ModelProviderConfigValidator defines the interface for validating model provider configurations
type ModelProviderConfigValidator interface {
	ValidateConfig(config map[string]string) error
	GetProviderName() string
}

// BaseValidator provides common validation functionality
type BaseValidator struct {
	providerName string
}

// ModelProvider is a struct that contains the model provider
type ModelProvider struct {
	Provider  string
	Validator ModelProviderConfigValidator
}

// NewModelProvider creates a new ModelProvider instance
func NewModelProvider(provider string) (*ModelProvider, error) {
	return &ModelProvider{
		Provider:  provider,
		Validator: NewBaseValidator(provider),
	}, nil
}

// GetProvider returns the provider name
func (m *ModelProvider) GetProvider() string {
	return m.Provider
}

// getRequiredFields returns the required fields for a given provider
func getRequiredFields(provider string) []string {
	if fields, ok := providerRequiredFields[provider]; ok {
		return fields
	}
	return DefaultRequiredFields
}

// NewBaseValidator creates a new base validator
func NewBaseValidator(providerName string) *BaseValidator {
	return &BaseValidator{
		providerName: providerName,
	}
}

// ValidateRequiredFields validates that all required fields are present and non-empty
func (b *BaseValidator) ValidateRequiredFields(config map[string]string) error {
	requiredFields := getRequiredFields(b.providerName)

	for _, field := range requiredFields {
		value, exists := config[field]
		if !exists {
			return fmt.Errorf("required field '%s' is missing for %s provider", field, b.providerName)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("required field '%s' cannot be empty for %s provider", field, b.providerName)
		}
	}

	return nil
}

// ValidateOptionalStringField validates that an optional string field is not empty if present
func (b *BaseValidator) ValidateOptionalStringField(config map[string]string, fieldName string) error {
	if value, exists := config[fieldName]; exists {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("field '%s' cannot be empty if provided for %s provider", fieldName, b.providerName)
		}
	}
	return nil
}

func (b *BaseValidator) ValidateConfig(config map[string]string) error {
	return b.ValidateRequiredFields(config)
}

func (b *BaseValidator) GetProviderName() string {
	return b.providerName
}

// ValidateConfig validates the configuration for the current provider
func (m *ModelProvider) ValidateConfig(config map[string]string) error {
	return m.Validator.ValidateConfig(config)
}
