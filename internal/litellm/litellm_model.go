package litellm

import (
	"context"
	"encoding/json"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmModel interface {
	CreateModel(ctx context.Context, req *ModelRequest) (ModelResponse, error)
	DeleteModel(ctx context.Context, modelName string) error
	GetModel(ctx context.Context, modelName string) (ModelResponse, error)
	IsModelUpdateNeeded(ctx context.Context, model *ModelResponse, req *ModelRequest) bool
	UpdateModel(ctx context.Context, req *ModelRequest) (ModelResponse, error)
}

// ModelRequest represents the request structure for creating/updating models
type ModelRequest struct {
	ModelName     string               `json:"model_name"`
	LiteLLMParams *UpdateLiteLLMParams `json:"litellm_params,omitempty"`
	ModelInfo     *ModelInfo           `json:"model_info,omitempty"`
}

// ModelResponse represents the response structure for model operations
type ModelResponse struct {
	ModelName     string               `json:"model_name"`
	LiteLLMParams *UpdateLiteLLMParams `json:"litellm_params,omitempty"`
	ModelInfo     *ModelInfo           `json:"model_info,omitempty"`
}

// UpdateLiteLLMParams represents the LiteLLM parameters for model configuration
type UpdateLiteLLMParams struct {
	InputCostPerToken                *float64               `json:"input_cost_per_token,omitempty"`
	OutputCostPerToken               *float64               `json:"output_cost_per_token,omitempty"`
	InputCostPerSecond               *float64               `json:"input_cost_per_second,omitempty"`
	OutputCostPerSecond              *float64               `json:"output_cost_per_second,omitempty"`
	InputCostPerPixel                *float64               `json:"input_cost_per_pixel,omitempty"`
	OutputCostPerPixel               *float64               `json:"output_cost_per_pixel,omitempty"`
	ApiKey                           *string                `json:"api_key,omitempty"`
	ApiBase                          *string                `json:"api_base,omitempty"`
	ApiVersion                       *string                `json:"api_version,omitempty"`
	VertexProject                    *string                `json:"vertex_project,omitempty"`
	VertexLocation                   *string                `json:"vertex_location,omitempty"`
	VertexCredentials                interface{}            `json:"vertex_credentials,omitempty"`
	RegionName                       *string                `json:"region_name,omitempty"`
	AwsAccessKeyID                   *string                `json:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey               *string                `json:"aws_secret_access_key,omitempty"`
	AwsRegionName                    *string                `json:"aws_region_name,omitempty"`
	WatsonXRegionName                *string                `json:"watsonx_region_name,omitempty"`
	CustomLLMProvider                *string                `json:"custom_llm_provider,omitempty"`
	TPM                              *int                   `json:"tpm,omitempty"`
	RPM                              *int                   `json:"rpm,omitempty"`
	Timeout                          interface{}            `json:"timeout,omitempty"`
	StreamTimeout                    interface{}            `json:"stream_timeout,omitempty"`
	MaxRetries                       *int                   `json:"max_retries,omitempty"`
	Organization                     *string                `json:"organization,omitempty"`
	ConfigurableClientsideAuthParams []interface{}          `json:"configurable_clientside_auth_params,omitempty"`
	LiteLLMCredentialName            *string                `json:"litellm_credential_name,omitempty"`
	LiteLLMTraceID                   *string                `json:"litellm_trace_id,omitempty"`
	MaxFileSizeMB                    *int                   `json:"max_file_size_mb,omitempty"`
	MaxBudget                        *float64               `json:"max_budget,omitempty"`
	BudgetDuration                   *string                `json:"budget_duration,omitempty"`
	UseInPassThrough                 *bool                  `json:"use_in_pass_through,omitempty"`
	UseLiteLLMProxy                  *bool                  `json:"use_litellm_proxy,omitempty"`
	MergeReasoningContentInChoices   *bool                  `json:"merge_reasoning_content_in_choices,omitempty"`
	ModelInfo                        interface{}            `json:"model_info,omitempty"`
	MockResponse                     interface{}            `json:"mock_response,omitempty"`
	AutoRouterConfigPath             *string                `json:"auto_router_config_path,omitempty"`
	AutoRouterConfig                 *string                `json:"auto_router_config,omitempty"`
	AutoRouterDefaultModel           *string                `json:"auto_router_default_model,omitempty"`
	AutoRouterEmbeddingModel         *string                `json:"auto_router_embedding_model,omitempty"`
	Model                            *string                `json:"model,omitempty"`
	AdditionalProperties             map[string]interface{} `json:"-"`
}

// ModelInfo represents the model information structure
type ModelInfo struct {
	ID                   *string                `json:"id,omitempty"`
	DBModel              *bool                  `json:"db_model,omitempty"`
	UpdatedAt            *string                `json:"updated_at,omitempty"`
	UpdatedBy            *string                `json:"updated_by,omitempty"`
	CreatedAt            *string                `json:"created_at,omitempty"`
	CreatedBy            *string                `json:"created_by,omitempty"`
	BaseModel            *string                `json:"base_model,omitempty"`
	Tier                 *string                `json:"tier,omitempty"`
	TeamID               *string                `json:"team_id,omitempty"`
	TeamPublicModelName  *string                `json:"team_public_model_name,omitempty"`
	AdditionalProperties map[string]interface{} `json:"-"`
}

func NewModelInfo() *ModelInfo {
	dbModel := true
	return &ModelInfo{
		DBModel: &dbModel,
	}
}

// CreateModel creates a new model in the LiteLLM service
func (l *LitellmClient) CreateModel(ctx context.Context, req *ModelRequest) (ModelResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal model request payload")
		return ModelResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/model/new", body)
	if err != nil {
		log.Error(err, "Failed to create model in LiteLLM")
		return ModelResponse{}, err
	}

	var modelResponse ModelResponse
	if err := json.Unmarshal(response, &modelResponse); err != nil {
		log.Error(err, "Failed to unmarshal model response from LiteLLM")
		return ModelResponse{}, err
	}

	return modelResponse, nil
}

// UpdateModel updates an existing model in the LiteLLM service
func (l *LitellmClient) UpdateModel(ctx context.Context, req *ModelRequest) (ModelResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal model update request payload")
		return ModelResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/model/update", body)
	if err != nil {
		log.Error(err, "Failed to update model in LiteLLM")
		return ModelResponse{}, err
	}

	var modelResponse ModelResponse
	if err := json.Unmarshal(response, &modelResponse); err != nil {
		log.Error(err, "Failed to unmarshal model response from LiteLLM")
		return ModelResponse{}, err
	}

	return modelResponse, nil
}

// GetModel retrieves a model from the LiteLLM service
func (l *LitellmClient) GetModel(ctx context.Context, modelID string) (ModelResponse, error) {
	log := log.FromContext(ctx)

	response, err := l.makeRequest(ctx, "GET", "/model/?litellm_model_id="+modelID, nil)
	if err != nil {
		log.Error(err, "Failed to get model from LiteLLM")
		return ModelResponse{}, err
	}

	var modelResponse ModelResponse
	if err := json.Unmarshal(response, &modelResponse); err != nil {
		log.Error(err, "Failed to unmarshal model response from LiteLLM")
		return ModelResponse{}, err
	}

	return modelResponse, nil
}

// DeleteModel deletes a model from the LiteLLM service
func (l *LitellmClient) DeleteModel(ctx context.Context, modelId string) error {
	log := log.FromContext(ctx)

	body := []byte(`{"id": "` + modelId + `"}`)

	if _, err := l.makeRequest(ctx, "POST", "/model/delete", body); err != nil {
		log.Error(err, "Failed to delete model from LiteLLM")
		return nil
	}

	return nil
}

// IsModelUpdateNeeded checks if the model needs to be updated
func (l *LitellmClient) IsModelUpdateNeeded(ctx context.Context, model *ModelResponse, req *ModelRequest) bool {
	log := log.FromContext(ctx)

	// Compare model names
	if model.ModelName != req.ModelName {
		log.Info("Model name changed")
		return true
	}

	// Compare LiteLLM parameters
	if !cmp.Equal(model.LiteLLMParams, req.LiteLLMParams, cmpopts.EquateEmpty()) {
		log.Info("LiteLLM parameters changed")
		return true
	}

	// Compare model info
	if !cmp.Equal(model.ModelInfo, req.ModelInfo, cmpopts.EquateEmpty()) {
		log.Info("Model info changed")
		return true
	}

	return false
}
