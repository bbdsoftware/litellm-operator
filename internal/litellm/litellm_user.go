package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmUser interface {
	CreateUser(ctx context.Context, req *UserRequest) (UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	CheckUserExists(ctx context.Context, userEmail string) (bool, error)
}

type UserRequest struct {
	Aliases              map[string]string `json:"aliases,omitempty"`
	AllowedCacheControls []string          `json:"allowed_cache_controls,omitempty"`
	AutoCreateKey        bool              `json:"auto_create_key,omitempty"`
	Blocked              bool              `json:"blocked,omitempty"`
	BudgetDuration       string            `json:"budget_duration,omitempty"`
	Config               map[string]string `json:"config,omitempty"`
	Duration             string            `json:"duration,omitempty"`
	Guardrails           []string          `json:"guardrails,omitempty"`
	KeyAlias             string            `json:"key_alias,omitempty"`
	MaxBudget            float64           `json:"max_budget,omitempty"`
	MaxParallelRequests  int               `json:"max_parallel_requests,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	ModelMaxBudget       map[string]string `json:"model_max_budget,omitempty"`
	ModelRPMLimit        map[string]string `json:"model_rpm_limit,omitempty"`
	ModelTPMLimit        map[string]string `json:"model_tpm_limit,omitempty"`
	Models               []string          `json:"models,omitempty"`
	Permissions          map[string]string `json:"permissions,omitempty"`
	RPMLimit             int               `json:"rpm_limit,omitempty"`
	SendInviteEmail      bool              `json:"send_invite_email,omitempty"`
	SoftBudget           float64           `json:"soft_budget,omitempty"`
	SSOUserID            string            `json:"sso_user_id,omitempty"`
	Teams                []string          `json:"teams,omitempty"`
	TPMLimit             int               `json:"tpm_limit,omitempty"`
	UserAlias            string            `json:"user_alias,omitempty"`
	UserEmail            string            `json:"user_email,omitempty"`
	UserID               string            `json:"user_id,omitempty"`
	UserRole             string            `json:"user_role,omitempty"`
}

type UserResponse struct {
	Aliases              map[string]string `json:"aliases,omitempty"`
	AllowedCacheControls []string          `json:"allowed_cache_controls,omitempty"`
	AllowedRoutes        []string          `json:"allowed_routes,omitempty"`
	AutoCreateKey        bool              `json:"auto_create_key,omitempty"`
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
	SendInviteEmail      bool              `json:"send_invite_email,omitempty"`
	Spend                float64           `json:"spend,omitempty"`
	SSOUserID            string            `json:"sso_user_id,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Teams                []string          `json:"teams,omitempty"`
	Token                string            `json:"token,omitempty"`
	TPMLimit             int               `json:"tpm_limit,omitempty"`
	UpdatedAt            string            `json:"updated_at,omitempty"`
	UpdatedBy            string            `json:"updated_by,omitempty"`
	UserAlias            string            `json:"user_alias,omitempty"`
	UserEmail            string            `json:"user_email,omitempty"`
	UserID               string            `json:"user_id,omitempty"`
	UserRole             string            `json:"user_role,omitempty"`
}

// CreateUser creates a new user in the Litellm service
func (l *LitellmClient) CreateUser(ctx context.Context, req *UserRequest) (UserResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal user request payload")
		return UserResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/user/new", body)
	if err != nil {
		log.Error(err, "Failed to create user in Litellm")
		return UserResponse{}, err
	}

	// convert response to CreateUserResponse
	var createUserResponse UserResponse
	if err := json.Unmarshal(response, &createUserResponse); err != nil {
		log.Error(err, "Failed to unmarshal create user response from Litellm")
		return UserResponse{}, err
	}

	return createUserResponse, nil
}

// DeleteUser deletes a user from the Litellm service
func (l *LitellmClient) DeleteUser(ctx context.Context, userID string) error {
	log := log.FromContext(ctx)

	body := []byte(`{"user_ids": ["` + userID + `"]}`)

	if _, err := l.makeRequest(ctx, "POST", "/user/delete", body); err != nil {
		log.Error(err, "Failed to delete user in Litellm")
		return err
	}

	return nil
}

// CheckUserExists checks if a user already exists in the Litellm service
func (l *LitellmClient) CheckUserExists(ctx context.Context, userEmail string) (bool, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/user/list?user_email="+userEmail, nil)
	if err != nil {
		log.Error(err, "Failed to check if User exists")
		return false, err
	}

	var response struct {
		Users []struct {
			UserID    string `json:"user_id"`
			UserEmail string `json:"user_email"`
		} `json:"users"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return false, err
	}

	// Check if any user exists with the given email
	// Since emails are unique, we only need to check the first user if any exists
	return len(response.Users) > 0 && response.Users[0].UserEmail == userEmail, nil
}
