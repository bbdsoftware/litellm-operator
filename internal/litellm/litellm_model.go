package litellm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmModel interface {
	CreateModel(ctx context.Context, req *ModelRequest) (ModelResponse, error)
	DeleteModel(ctx context.Context, modelName string) error
	GetModel(ctx context.Context, modelName string) (ModelResponse, error)
	GetModelInfo(ctx context.Context, modelID string) (ModelResponse, error)
	IsModelUpdateNeeded(ctx context.Context, model *ModelResponse, req *ModelRequest) (ModelUpdateNeeded, error)
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

type ModelListResponse struct {
	Data []ModelResponse `json:"data"`
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

func (l *LitellmClient) GetModelInfo(ctx context.Context, modelID string) (ModelResponse, error) {
	log := log.FromContext(ctx)

	response, err := l.makeRequest(ctx, "GET", "/model/info?litellm_model_id="+modelID, nil)
	if err != nil {
		log.Error(err, "Failed to get model info from LiteLLM")
		return ModelResponse{}, err
	}

	var modelListResponse ModelListResponse
	if err := json.Unmarshal(response, &modelListResponse); err != nil {
		log.Error(err, "Failed to unmarshal model response from LiteLLM")
		return ModelResponse{}, err
	}

	return modelListResponse.Data[0], nil
}

// GetModel retrieves a model from the LiteLLM service
func (l *LitellmClient) GetModel(ctx context.Context, modelID string) (ModelResponse, error) {
	log := log.FromContext(ctx)

	response, err := l.makeRequest(ctx, "GET", "/model?litellm_model_id="+modelID, nil)
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
		return err
	}

	return nil
}

type ModelUpdateNeeded struct {
	NeedsUpdate   bool
	ChangedFields []FieldChange
}

// IsModelUpdateNeeded checks if the model needs to be updated
func (l *LitellmClient) IsModelUpdateNeeded(ctx context.Context, model *ModelResponse, req *ModelRequest) (ModelUpdateNeeded, error) {
	log := log.FromContext(ctx)
	var changedFields ModelUpdateNeeded
	// Helper function to check field changes
	checkField := func(fieldName, logName string, current, expected interface{}, equateEmpty bool, needsUpdate bool) {
		var changed bool
		if equateEmpty {
			changed = !cmp.Equal(current, expected, cmpopts.EquateEmpty())
		} else {
			changed = !reflect.DeepEqual(current, expected)
		}

		if changed {
			log.Info(fmt.Sprintf("%s changed", logName))
			if needsUpdate {
				changedFields.NeedsUpdate = true
			}
			changedFields.ChangedFields = append(changedFields.ChangedFields, FieldChange{
				FieldName:     fieldName,
				CurrentValue:  current,
				ExpectedValue: expected,
			})
		}
	}

	checkField("model_name", "Model name", model.ModelName, req.ModelName, true, true)
	checkField("litellm_params", "LiteLLM params", model.LiteLLMParams, req.LiteLLMParams, true, true)
	checkField("model_info", "Model info", model.ModelInfo, req.ModelInfo, true, true)

	// Example of more granular LiteLLMParams field checking
	if model.LiteLLMParams != nil && req.LiteLLMParams != nil {
		checkField("input_cost_per_token", "Input cost per token", model.LiteLLMParams.InputCostPerToken, req.LiteLLMParams.InputCostPerToken, true, true)
		checkField("output_cost_per_token", "Output cost per token", model.LiteLLMParams.OutputCostPerToken, req.LiteLLMParams.OutputCostPerToken, true, true)

	}

	// Example of more granular ModelInfo field checking
	if model.ModelInfo != nil && req.ModelInfo != nil {
		checkField("model_info_id", "Model info ID", model.ModelInfo.ID, req.ModelInfo.ID, true, true)
		checkField("team_id", "Team ID", model.ModelInfo.TeamID, req.ModelInfo.TeamID, true, true)
		checkField("team_public_model_name", "Team public model name", model.ModelInfo.TeamPublicModelName, req.ModelInfo.TeamPublicModelName, true, true)
	}

	if changedFields.NeedsUpdate {
		log.Info("Model update needed")
		for _, field := range changedFields.ChangedFields {
			log.Info(fmt.Sprintf("Field changed: %s", field.FieldName), "current", field.CurrentValue, "expected", field.ExpectedValue)
		}

	}

	return changedFields, nil
}
