package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmVirtualKey interface {
	GenerateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error)
	DeleteVirtualKey(ctx context.Context, keyAlias string) error
	CheckVirtualKeyExists(ctx context.Context, keyAlias string) (bool, error)
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
		Keys []struct {
			KeyAlias string `json:"key_alias"`
			KeyName  string `json:"key_name"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return false, err
	}

	// Check if any key exists with the given alias
	// Since key aliases are unique, we only need to check the first key if any exists
	return len(response.Keys) > 0 && response.Keys[0].KeyAlias == keyAlias, nil
}
