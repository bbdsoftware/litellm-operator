package litellm

import (
	"context"
	"encoding/json"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmVirtualKey interface {
	CheckVirtualKeyExists(ctx context.Context, keyAlias string) (bool, error)
	DeleteVirtualKey(ctx context.Context, keyAlias string) error
	GenerateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error)
	GetVirtualKey(ctx context.Context, key string) (VirtualKeyResponse, error)
	UpdateNeeded(ctx context.Context, virtualKey *VirtualKeyResponse, req *VirtualKeyRequest) bool
	UpdateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error)
}

type VirtualKeyRequest struct {
	Aliases              map[string]string `json:"aliases,omitempty"`
	AllowedCacheControls []string          `json:"allowed_cache_controls,omitempty"`
	AllowedRoutes        []string          `json:"allowed_routes,omitempty"`
	Blocked              bool              `json:"blocked,omitempty"`
	BudgetDuration       string            `json:"budget_duration,omitempty"`
	BudgetID             string            `json:"budget_id,omitempty"`
	Config               map[string]string `json:"config,omitempty"`
	Duration             string            `json:"duration,omitempty"`
	EnforcedParams       []string          `json:"enforced_params,omitempty"`
	Guardrails           []string          `json:"guardrails,omitempty"`
	Key                  string            `json:"key,omitempty"`
	KeyAlias             string            `json:"key_alias,omitempty"`
	MaxBudget            float64           `json:"max_budget,omitempty"`
	MaxParallelRequests  int               `json:"max_parallel_requests,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	ModelMaxBudget       map[string]string `json:"model_max_budget,omitempty"`
	ModelRPMLimit        map[string]int    `json:"model_rpm_limit,omitempty"`
	ModelTPMLimit        map[string]int    `json:"model_tpm_limit,omitempty"`
	Models               []string          `json:"models,omitempty"`
	Permissions          map[string]string `json:"permissions,omitempty"`
	RPMLimit             int               `json:"rpm_limit,omitempty"`
	SendInviteEmail      bool              `json:"send_invite_email,omitempty"`
	SoftBudget           float64           `json:"soft_budget,omitempty"`
	Spend                float64           `json:"spend,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	TeamID               string            `json:"team_id,omitempty"`
	TPMLimit             int               `json:"tpm_limit,omitempty"`
	UserID               string            `json:"user_id,omitempty"`
}

type VirtualKeyResponse struct {
	Aliases              map[string]string `json:"aliases,omitempty"`
	AllowedCacheControls []string          `json:"allowed_cache_controls,omitempty"`
	AllowedRoutes        []string          `json:"allowed_routes,omitempty"`
	Blocked              bool              `json:"blocked,omitempty"`
	BudgetDuration       string            `json:"budget_duration,omitempty"`
	BudgetID             string            `json:"budget_id,omitempty"`
	BudgetResetAt        string            `json:"budget_reset_at,omitempty"`
	Config               map[string]string `json:"config,omitempty"`
	CreatedAt            string            `json:"created_at,omitempty"`
	CreatedBy            string            `json:"created_by,omitempty"`
	Duration             string            `json:"duration,omitempty"`
	EnforcedParams       []string          `json:"enforced_params,omitempty"`
	Expires              string            `json:"expires,omitempty"`
	Guardrails           []string          `json:"guardrails,omitempty"`
	Key                  string            `json:"key,omitempty"`
	KeyAlias             string            `json:"key_alias,omitempty"`
	KeyName              string            `json:"key_name,omitempty"`
	LiteLLMBudgetTable   string            `json:"litellm_budget_table,omitempty"`
	MaxBudget            float64           `json:"max_budget,omitempty"`
	MaxParallelRequests  int               `json:"max_parallel_requests,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	ModelMaxBudget       map[string]string `json:"model_max_budget,omitempty"`
	ModelRPMLimit        map[string]string `json:"model_rpm_limit,omitempty"`
	ModelTPMLimit        map[string]string `json:"model_tpm_limit,omitempty"`
	Models               []string          `json:"models,omitempty"`
	Permissions          map[string]string `json:"permissions,omitempty"`
	RPMLimit             int               `json:"rpm_limit,omitempty"`
	Spend                float64           `json:"spend,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	TeamID               string            `json:"team_id,omitempty"`
	Token                string            `json:"token,omitempty"`
	TokenID              string            `json:"token_id,omitempty"`
	TPMLimit             int               `json:"tpm_limit,omitempty"`
	UpdatedAt            string            `json:"updated_at,omitempty"`
	UpdatedBy            string            `json:"updated_by,omitempty"`
	UserID               string            `json:"user_id,omitempty"`
}

// GenerateVirtualKey generates a new virtual key for the Litellm service
func (l *LitellmClient) GenerateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal virtual key request payload")
		return VirtualKeyResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/key/generate", body)
	if err != nil {
		log.Error(err, "Failed to create virtual key in Litellm")
		return VirtualKeyResponse{}, err
	}

	var virtualKeyResponse VirtualKeyResponse
	if err := json.Unmarshal(response, &virtualKeyResponse); err != nil {
		log.Error(err, "Failed to unmarshal virtual key response from Litellm")
		return VirtualKeyResponse{}, err
	}

	return virtualKeyResponse, nil
}

func (l *LitellmClient) UpdateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal virtual key request payload")
		return VirtualKeyResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/key/update", body)
	if err != nil {
		log.Error(err, "Failed to update virtual key in Litellm")
		return VirtualKeyResponse{}, err
	}

	var virtualKeyResponse VirtualKeyResponse
	if err := json.Unmarshal(response, &virtualKeyResponse); err != nil {
		log.Error(err, "Failed to unmarshal virtual key response from Litellm")
		return VirtualKeyResponse{}, err
	}

	return virtualKeyResponse, nil
}

// DeleteVirtualKey deletes a virtual key from the Litellm service
func (l *LitellmClient) DeleteVirtualKey(ctx context.Context, keyAlias string) error {
	log := log.FromContext(ctx)

	body := []byte(`{"key_aliases": ["` + keyAlias + `"]}`)

	if _, err := l.makeRequest(ctx, "POST", "/key/delete", body); err != nil {
		log.Error(err, "Failed to delete virtual key in Litellm")
		return err
	}

	return nil
}

// CheckVirtualKeyExists checks if a virtual key already exists in the Litellm service
func (l *LitellmClient) CheckVirtualKeyExists(ctx context.Context, keyAlias string) (bool, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/key/list?key_alias="+keyAlias, nil)
	if err != nil {
		log.Error(err, "Failed to check if virtual key exists")
		return false, err
	}

	var response struct {
		Keys []string `json:"keys"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return false, err
	}

	// Check if any key exists with the given alias
	if len(response.Keys) == 0 {
		return false, nil
	}
	return true, nil
}

// GetVirtualKey gets a virtual key from the Litellm service
func (l *LitellmClient) GetVirtualKey(ctx context.Context, key string) (VirtualKeyResponse, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/key/info?key="+key, nil)
	if err != nil {
		log.Error(err, "Failed to get virtual key")
		return VirtualKeyResponse{}, err
	}

	var response struct {
		KeyInfo VirtualKeyResponse `json:"info"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal virtual key response from Litellm")
		return VirtualKeyResponse{}, err
	}

	return response.KeyInfo, nil
}

// UpdateNeeded checks if the virtual key needs to be updated
func (l *LitellmClient) UpdateNeeded(ctx context.Context, virtualKey *VirtualKeyResponse, req *VirtualKeyRequest) bool {
	log := log.FromContext(ctx)

	// Helper function to compare slices, treating nil and empty as equivalent
	sliceChanged := func(virtual, request interface{}) bool {
		v := reflect.ValueOf(virtual)
		r := reflect.ValueOf(request)

		// Handle nil cases
		if v.IsNil() && r.IsNil() {
			return false
		}
		if v.IsNil() {
			return r.Len() > 0
		}
		if r.IsNil() {
			return v.Len() > 0
		}

		// If both are non-nil, compare lengths first
		if v.Len() == 0 && r.Len() == 0 {
			return false
		}

		// If both have elements, do deep comparison
		return !reflect.DeepEqual(virtual, request)
	}

	if sliceChanged(virtualKey.Aliases, req.Aliases) {
		log.Info("Aliases changed")
		return true
	}
	if sliceChanged(virtualKey.AllowedCacheControls, req.AllowedCacheControls) {
		log.Info("AllowedCacheControls changed")
		return true
	}
	if sliceChanged(virtualKey.AllowedRoutes, req.AllowedRoutes) {
		log.Info("AllowedRoutes changed")
		return true
	}
	if virtualKey.Blocked != req.Blocked {
		log.Info("Blocked changed")
		return true
	}
	if virtualKey.BudgetDuration != req.BudgetDuration {
		log.Info("BudgetDuration changed")
		return true
	}
	if virtualKey.BudgetID != req.BudgetID {
		log.Info("BudgetID changed")
		return true
	}
	if sliceChanged(virtualKey.Config, req.Config) {
		log.Info("Config changed")
		return true
	}
	if virtualKey.Duration != req.Duration {
		log.Info("Duration changed")
		return true
	}
	if sliceChanged(virtualKey.EnforcedParams, req.EnforcedParams) {
		log.Info("EnforcedParams changed")
		return true
	}
	if sliceChanged(virtualKey.Guardrails, req.Guardrails) {
		log.Info("Guardrails changed")
		return true
	}
	if virtualKey.KeyAlias != req.KeyAlias {
		log.Info("KeyAlias changed")
		return true
	}
	if virtualKey.MaxBudget != req.MaxBudget {
		log.Info("MaxBudget changed")
		return true
	}
	if virtualKey.MaxParallelRequests != req.MaxParallelRequests {
		log.Info("MaxParallelRequests changed")
		return true
	}
	if sliceChanged(virtualKey.Metadata, req.Metadata) {
		log.Info("Metadata changed")
		return true
	}
	if sliceChanged(virtualKey.ModelMaxBudget, req.ModelMaxBudget) {
		log.Info("ModelMaxBudget changed")
		return true
	}
	if sliceChanged(virtualKey.ModelRPMLimit, req.ModelRPMLimit) {
		log.Info("ModelRPMLimit changed")
		return true
	}
	if sliceChanged(virtualKey.ModelTPMLimit, req.ModelTPMLimit) {
		log.Info("ModelTPMLimit changed")
		return true
	}
	if sliceChanged(virtualKey.Models, req.Models) {
		log.Info("Models changed")
		return true
	}
	if sliceChanged(virtualKey.Permissions, req.Permissions) {
		log.Info("Permissions changed")
		return true
	}
	if virtualKey.RPMLimit != req.RPMLimit {
		log.Info("RPMLimit changed")
		return true
	}
	if sliceChanged(virtualKey.Tags, req.Tags) {
		log.Info("Tags changed")
		return true
	}
	if virtualKey.TeamID != req.TeamID {
		log.Info("TeamID changed")
		return true
	}
	if virtualKey.TPMLimit != req.TPMLimit {
		log.Info("TPMLimit changed")
		return true
	}
	if virtualKey.UserID != req.UserID {
		log.Info("UserID changed")
		return true
	}
	return false
}
