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

type LitellmUser interface {
	CreateUser(ctx context.Context, req *UserRequest) (UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	GetUser(ctx context.Context, userID string) (UserResponse, error)
	GetUserID(ctx context.Context, userEmail string) (string, error)
	GetTeam(ctx context.Context, teamID string) (TeamResponse, error)
	IsUserUpdateNeeded(ctx context.Context, user *UserResponse, req *UserRequest) (UserUpdateNeeded, error)
	UpdateUser(ctx context.Context, req *UserRequest) (UserResponse, error)
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
	// These don't actually come back here, they are injected into the metadata field which complicates things, so skip for now
	// ModelMaxBudget       map[string]string `json:"model_max_budget,omitempty"`
	// ModelRPMLimit        map[string]string `json:"model_rpm_limit,omitempty"`
	// ModelTPMLimit        map[string]string `json:"model_tpm_limit,omitempty"`
	Models          []string          `json:"models,omitempty"`
	Permissions     map[string]string `json:"permissions,omitempty"`
	RPMLimit        int               `json:"rpm_limit,omitempty"`
	SendInviteEmail bool              `json:"send_invite_email,omitempty"`
	Spend           float64           `json:"spend,omitempty"`
	SSOUserID       string            `json:"sso_user_id,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Teams           []string          `json:"teams,omitempty"`
	Token           string            `json:"token,omitempty"`
	TPMLimit        int               `json:"tpm_limit,omitempty"`
	UpdatedAt       string            `json:"updated_at,omitempty"`
	UpdatedBy       string            `json:"updated_by,omitempty"`
	UserAlias       string            `json:"user_alias,omitempty"`
	UserEmail       string            `json:"user_email,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
	UserRole        string            `json:"user_role,omitempty"`
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

	// convert response to UserResponse
	var userResponse UserResponse
	if err := json.Unmarshal(response, &userResponse); err != nil {
		log.Error(err, "Failed to unmarshal create user response from Litellm")
		return UserResponse{}, err
	}

	return userResponse, nil
}

// UpdateUser updates an existing user in the Litellm service
func (l *LitellmClient) UpdateUser(ctx context.Context, req *UserRequest) (UserResponse, error) {
	log := log.FromContext(ctx)

	// These are not implemented in the update endpoint
	req.Duration = ""
	req.KeyAlias = ""

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal user request payload")
		return UserResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/user/update", body)
	if err != nil {
		log.Error(err, "Failed to update user in Litellm")
		return UserResponse{}, err
	}

	var responseBody struct {
		User UserResponse `json:"data"`
	}

	if err := json.Unmarshal(response, &responseBody); err != nil {
		log.Error(err, "Failed to unmarshal update user response from Litellm")
		return UserResponse{}, err
	}

	return responseBody.User, nil
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

// GetUserID gets the ID of a user from the Litellm service, returns empty string if user email not found
func (l *LitellmClient) GetUserID(ctx context.Context, userEmail string) (string, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/user/list?user_email="+userEmail, nil)
	if err != nil {
		log.Error(err, "Failed to check if User exists")
		return "", err
	}

	var response struct {
		Users []struct {
			UserID    string `json:"user_id"`
			UserEmail string `json:"user_email"`
		} `json:"users"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return "", err
	}

	// Check if any user exists with the given email
	// Since emails are unique, we only need to check the first user if any exists
	if len(response.Users) > 0 && response.Users[0].UserEmail == userEmail {
		return response.Users[0].UserID, nil
	}

	return "", nil
}

// GetUser gets a user from the Litellm service
func (l *LitellmClient) GetUser(ctx context.Context, userID string) (UserResponse, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/user/info?user_id="+userID, nil)
	if err != nil {
		log.Error(err, "Failed to get user")
		return UserResponse{}, err
	}

	var response struct {
		User UserResponse `json:"user_info"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal user response from Litellm")
		return UserResponse{}, err
	}

	return response.User, nil
}

// IsUserUpdateNeeded checks if a user needs to be updated in the Litellm service
// Returns a boolean indicating if an update is needed and a slice of field names that have changed

type FieldChange struct {
	FieldName     string
	CurrentValue  interface{}
	ExpectedValue interface{}
}

type UserUpdateNeeded struct {
	NeedsUpdate   bool
	ChangedFields []FieldChange
}

func (l *LitellmClient) IsUserUpdateNeeded(ctx context.Context, user *UserResponse, req *UserRequest) (UserUpdateNeeded, error) {
	log := log.FromContext(ctx)
	var changedFields UserUpdateNeeded

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

	// Check all fields using the helper
	checkField("aliases", "Aliases", user.Aliases, req.Aliases, true, false)
	checkField("allowed_cache_controls", "AllowedCacheControls", user.AllowedCacheControls, req.AllowedCacheControls, true, false)
	checkField("blocked", "Blocked", user.Blocked, req.Blocked, false, false)
	checkField("budget_duration", "BudgetDuration", user.BudgetDuration, req.BudgetDuration, false, false)
	checkField("config", "Config", user.Config, req.Config, true, false)
	checkField("duration", "Duration", user.Duration, req.Duration, false, true)
	checkField("guardrails", "Guardrails", user.Guardrails, req.Guardrails, true, true)
	checkField("max_budget", "MaxBudget", user.MaxBudget, req.MaxBudget, false, false)
	checkField("max_parallel_requests", "MaxParallelRequests", user.MaxParallelRequests, req.MaxParallelRequests, false, true)
	checkField("models", "Models", user.Models, req.Models, true, false)
	checkField("permissions", "Permissions", user.Permissions, req.Permissions, true, false)
	checkField("rpm_limit", "RPMLimit", user.RPMLimit, req.RPMLimit, false, true)
	checkField("send_invite_email", "SendInviteEmail", user.SendInviteEmail, req.SendInviteEmail, false, true)
	checkField("sso_user_id", "SSOUserID", user.SSOUserID, req.SSOUserID, false, true)
	checkField("teams", "Teams", user.Teams, req.Teams, true, false)
	checkField("tpm_limit", "TPMLimit", user.TPMLimit, req.TPMLimit, false, false)
	checkField("user_alias", "UserAlias", user.UserAlias, req.UserAlias, false, true)
	checkField("user_email", "UserEmail", user.UserEmail, req.UserEmail, false, true)
	checkField("user_role", "UserRole", user.UserRole, req.UserRole, false, true)

	return changedFields, nil
}
