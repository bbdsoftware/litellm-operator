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
	VertexCredentials                *string                `json:"vertex_credentials,omitempty"`
	RegionName                       *string                `json:"region_name,omitempty"`
	AwsAccessKeyID                   *string                `json:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey               *string                `json:"aws_secret_access_key,omitempty"`
	AwsRegionName                    *string                `json:"aws_region_name,omitempty"`
	WatsonXRegionName                *string                `json:"watsonx_region_name,omitempty"`
	CustomLLMProvider                *string                `json:"custom_llm_provider,omitempty"`
	TPM                              *int                   `json:"tpm,omitempty"`
	RPM                              *int                   `json:"rpm,omitempty"`
	Timeout                          *float64               `json:"timeout,omitempty"`
	StreamTimeout                    *float64               `json:"stream_timeout,omitempty"`
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

	path := "/model/" + *req.ModelInfo.ID + "/update"
	response, err := l.makeRequest(ctx, "PATCH", path, body)
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

	// If one of the LiteLLMParams is nil and the other is not, consider this a change.
	if (model.LiteLLMParams == nil) != (req.LiteLLMParams == nil) {
		checkField("litellm_params", "LiteLLM params", model.LiteLLMParams, req.LiteLLMParams, true, true)
	}

	// More concise granular LiteLLMParams checking using descriptor list.
	if model.LiteLLMParams != nil && req.LiteLLMParams != nil {
		type paramCheck struct {
			fieldName   string
			logName     string
			modelVal    func(*UpdateLiteLLMParams) interface{}
			reqVal      func(*UpdateLiteLLMParams) interface{}
			equateEmpty bool
			needsUpdate bool
		}

		checks := []paramCheck{
			{"input_cost_per_token", "Input cost per token", func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerToken }, func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerToken }, true, true},
			{"output_cost_per_token", "Output cost per token", func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerToken }, func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerToken }, true, true},
			{"input_cost_per_second", "Input cost per second", func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerSecond }, func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerSecond }, true, true},
			{"output_cost_per_second", "Output cost per second", func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerSecond }, func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerSecond }, true, true},
			{"input_cost_per_pixel", "Input cost per pixel", func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerPixel }, func(p *UpdateLiteLLMParams) interface{} { return p.InputCostPerPixel }, true, true},
			{"output_cost_per_pixel", "Output cost per pixel", func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerPixel }, func(p *UpdateLiteLLMParams) interface{} { return p.OutputCostPerPixel }, true, true},
			{"api_base", "API base", func(p *UpdateLiteLLMParams) interface{} { return p.ApiBase }, func(p *UpdateLiteLLMParams) interface{} { return p.ApiBase }, true, true},
			{"api_version", "API version", func(p *UpdateLiteLLMParams) interface{} { return p.ApiVersion }, func(p *UpdateLiteLLMParams) interface{} { return p.ApiVersion }, true, true},
			{"vertex_project", "Vertex project", func(p *UpdateLiteLLMParams) interface{} { return p.VertexProject }, func(p *UpdateLiteLLMParams) interface{} { return p.VertexProject }, true, true},
			{"vertex_location", "Vertex location", func(p *UpdateLiteLLMParams) interface{} { return p.VertexLocation }, func(p *UpdateLiteLLMParams) interface{} { return p.VertexLocation }, true, true},
			{"region_name", "Region name", func(p *UpdateLiteLLMParams) interface{} { return p.RegionName }, func(p *UpdateLiteLLMParams) interface{} { return p.RegionName }, true, true},
			{"aws_region_name", "AWS region name", func(p *UpdateLiteLLMParams) interface{} { return p.AwsRegionName }, func(p *UpdateLiteLLMParams) interface{} { return p.AwsRegionName }, true, true},
			{"watsonx_region_name", "WatsonX region name", func(p *UpdateLiteLLMParams) interface{} { return p.WatsonXRegionName }, func(p *UpdateLiteLLMParams) interface{} { return p.WatsonXRegionName }, true, true},
			{"custom_llm_provider", "Custom LLM provider", func(p *UpdateLiteLLMParams) interface{} { return p.CustomLLMProvider }, func(p *UpdateLiteLLMParams) interface{} { return p.CustomLLMProvider }, true, true},
			{"tpm", "TPM", func(p *UpdateLiteLLMParams) interface{} { return p.TPM }, func(p *UpdateLiteLLMParams) interface{} { return p.TPM }, true, true},
			{"rpm", "RPM", func(p *UpdateLiteLLMParams) interface{} { return p.RPM }, func(p *UpdateLiteLLMParams) interface{} { return p.RPM }, true, true},
			{"timeout", "Timeout", func(p *UpdateLiteLLMParams) interface{} { return p.Timeout }, func(p *UpdateLiteLLMParams) interface{} { return p.Timeout }, true, true},
			{"stream_timeout", "Stream timeout", func(p *UpdateLiteLLMParams) interface{} { return p.StreamTimeout }, func(p *UpdateLiteLLMParams) interface{} { return p.StreamTimeout }, true, true},
			{"max_retries", "Max retries", func(p *UpdateLiteLLMParams) interface{} { return p.MaxRetries }, func(p *UpdateLiteLLMParams) interface{} { return p.MaxRetries }, true, true},
			{"organization", "Organization", func(p *UpdateLiteLLMParams) interface{} { return p.Organization }, func(p *UpdateLiteLLMParams) interface{} { return p.Organization }, true, true},
			{"litellm_credential_name", "LiteLLM credential name", func(p *UpdateLiteLLMParams) interface{} { return p.LiteLLMCredentialName }, func(p *UpdateLiteLLMParams) interface{} { return p.LiteLLMCredentialName }, true, true},
			{"litellm_trace_id", "LiteLLM trace ID", func(p *UpdateLiteLLMParams) interface{} { return p.LiteLLMTraceID }, func(p *UpdateLiteLLMParams) interface{} { return p.LiteLLMTraceID }, true, true},
			{"max_file_size_mb", "Max file size MB", func(p *UpdateLiteLLMParams) interface{} { return p.MaxFileSizeMB }, func(p *UpdateLiteLLMParams) interface{} { return p.MaxFileSizeMB }, true, true},
			{"max_budget", "Max budget", func(p *UpdateLiteLLMParams) interface{} { return p.MaxBudget }, func(p *UpdateLiteLLMParams) interface{} { return p.MaxBudget }, true, true},
			{"budget_duration", "Budget duration", func(p *UpdateLiteLLMParams) interface{} { return p.BudgetDuration }, func(p *UpdateLiteLLMParams) interface{} { return p.BudgetDuration }, true, true},
			{"use_in_pass_through", "Use in pass-through", func(p *UpdateLiteLLMParams) interface{} { return p.UseInPassThrough }, func(p *UpdateLiteLLMParams) interface{} { return p.UseInPassThrough }, true, true},
			{"use_litellm_proxy", "Use LiteLLM proxy", func(p *UpdateLiteLLMParams) interface{} { return p.UseLiteLLMProxy }, func(p *UpdateLiteLLMParams) interface{} { return p.UseLiteLLMProxy }, true, true},
			{"merge_reasoning_content_in_choices", "Merge reasoning content in choices", func(p *UpdateLiteLLMParams) interface{} { return p.MergeReasoningContentInChoices }, func(p *UpdateLiteLLMParams) interface{} { return p.MergeReasoningContentInChoices }, true, true},
			{"auto_router_config_path", "Auto router config path", func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterConfigPath }, func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterConfigPath }, true, true},
			{"auto_router_config", "Auto router config", func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterConfig }, func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterConfig }, true, true},
			{"auto_router_default_model", "Auto router default model", func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterDefaultModel }, func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterDefaultModel }, true, true},
			{"auto_router_embedding_model", "Auto router embedding model", func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterEmbeddingModel }, func(p *UpdateLiteLLMParams) interface{} { return p.AutoRouterEmbeddingModel }, true, true},
			{"model", "Model", func(p *UpdateLiteLLMParams) interface{} { return p.Model }, func(p *UpdateLiteLLMParams) interface{} { return p.Model }, true, true},
		}

		for _, c := range checks {
			checkField(c.fieldName, c.logName, c.modelVal(model.LiteLLMParams), c.reqVal(req.LiteLLMParams), c.equateEmpty, c.needsUpdate)
		}
	}

	// If one of the ModelInfo is nil and the other is not, consider this a change.
	if (model.ModelInfo == nil) != (req.ModelInfo == nil) {
		checkField("model_info", "Model info", model.ModelInfo, req.ModelInfo, true, true)
	}

	// Example of more granular ModelInfo field checking when both are present
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
