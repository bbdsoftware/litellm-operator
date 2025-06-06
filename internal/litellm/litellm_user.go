package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmUser interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (CreateUserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	CheckUserExists(ctx context.Context, userEmail string) (bool, error)
}

type CreateUserRequest struct {
	UserID              string            `json:"user_id,omitempty"`
	UserAlias           string            `json:"user_alias,omitempty"`
	UserEmail           string            `json:"user_email,omitempty"`
	UserRole            string            `json:"user_role,omitempty"`
	SendInviteEmail     bool              `json:"send_invite_email,omitempty"`
	Teams               []string          `json:"teams,omitempty"`
	AutoCreateKey       bool              `json:"auto_create_key,omitempty"`
	KeyAlias            string            `json:"key_alias,omitempty"`
	SoftBudget          string            `json:"soft_budget,omitempty"`
	MaxBudget           string            `json:"max_budget,omitempty"`
	ModelMaxBudget      map[string]string `json:"model_max_budget,omitempty"`
	ModelRPMLimit       map[string]string `json:"model_rpm_limit,omitempty"`
	ModelTPMLimit       map[string]string `json:"model_tpm_limit,omitempty"`
	BudgetDuration      string            `json:"budget_duration,omitempty"`
	Models              []string          `json:"models,omitempty"`
	MaxParallelRequests int               `json:"max_parallel_requests,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
}

type CreateUserResponse struct {
	CreatedAt           string            `json:"created_at"`
	UpdatedAt           string            `json:"updated_at"`
	Expires             string            `json:"expires"`
	UserID              string            `json:"user_id"`
	UserAlias           string            `json:"user_alias"`
	UserEmail           string            `json:"user_email"`
	UserRole            string            `json:"user_role"`
	Key                 string            `json:"key"`
	Teams               []string          `json:"teams"`
	KeyAlias            string            `json:"key_alias"`
	MaxBudget           float64           `json:"max_budget"`
	ModelMaxBudget      map[string]string `json:"model_max_budget"`
	ModelRPMLimit       map[string]string `json:"model_rpm_limit"`
	ModelTPMLimit       map[string]string `json:"model_tpm_limit"`
	BudgetDuration      string            `json:"budget_duration"`
	Models              []string          `json:"models"`
	MaxParallelRequests int               `json:"max_parallel_requests"`
	Metadata            map[string]string `json:"metadata"`
}

// CreateUser creates a new user in the Litellm service
func (l *LitellmClient) CreateUser(ctx context.Context, req *CreateUserRequest) (CreateUserResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal user request payload")
		return CreateUserResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/user/new", body)
	if err != nil {
		log.Error(err, "Failed to create user in Litellm")
		return CreateUserResponse{}, err
	}

	// convert response to CreateUserResponse
	var createUserResponse CreateUserResponse
	if err := json.Unmarshal(response, &createUserResponse); err != nil {
		log.Error(err, "Failed to unmarshal create user response from Litellm")
		return CreateUserResponse{}, err
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
