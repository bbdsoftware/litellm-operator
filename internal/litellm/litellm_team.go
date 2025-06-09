package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmTeam interface {
	CreateTeam(ctx context.Context, req *TeamRequest) (TeamResponse, error)
	DeleteTeam(ctx context.Context, teamID string) error
	CheckTeamExists(ctx context.Context, teamAlias string) (bool, error)
}

type TeamMemberWithRole struct {
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	Role      string `json:"role,omitempty"`
}

type TeamRequest struct {
	Admins                []string             `json:"admins,omitempty"`
	Blocked               bool                 `json:"blocked,omitempty"`
	BudgetDuration        string               `json:"budget_duration,omitempty"`
	Guardrails            []string             `json:"guardrails,omitempty"`
	MaxBudget             float64              `json:"max_budget,omitempty"`
	Members               []string             `json:"members,omitempty"`
	MembersWithRole       []TeamMemberWithRole `json:"members_with_roles,omitempty"`
	Metadata              map[string]string    `json:"metadata,omitempty"`
	ModelAliases          map[string]string    `json:"model_aliases,omitempty"`
	Models                []string             `json:"models,omitempty"`
	OrganizationID        string               `json:"organization_id,omitempty"`
	RPMLimit              int                  `json:"rpm_limit,omitempty"`
	Tags                  []string             `json:"tags,omitempty"`
	TeamAlias             string               `json:"team_alias,omitempty"`
	TeamID                string               `json:"team_id,omitempty"`
	TeamMemberPermissions []string             `json:"team_member_permissions,omitempty"`
	TPMLimit              int                  `json:"tpm_limit,omitempty"`
}

type TeamResponse struct {
	Admins                []string             `json:"admins,omitempty"`
	Blocked               bool                 `json:"blocked,omitempty"`
	BudgetDuration        string               `json:"budget_duration,omitempty"`
	BudgetResetAt         string               `json:"budget_reset_at,omitempty"`
	CreatedAt             string               `json:"created_at,omitempty"`
	LiteLLMModelTable     string               `json:"litellm_model_table,omitempty"`
	MaxBudget             float64              `json:"max_budget,omitempty"`
	MaxParallelRequests   int                  `json:"max_parallel_requests,omitempty"`
	Members               []string             `json:"members,omitempty"`
	MembersWithRole       []TeamMemberWithRole `json:"members_with_roles,omitempty"`
	Metadata              map[string]string    `json:"metadata,omitempty"`
	ModelID               string               `json:"model_id,omitempty"`
	Models                []string             `json:"models,omitempty"`
	OrganizationID        string               `json:"organization_id,omitempty"`
	RPMLimit              int                  `json:"rpm_limit,omitempty"`
	Spend                 float64              `json:"spend,omitempty"`
	Tags                  []string             `json:"tags,omitempty"`
	TeamAlias             string               `json:"team_alias,omitempty"`
	TeamID                string               `json:"team_id,omitempty"`
	TeamMemberPermissions []string             `json:"team_member_permissions,omitempty"`
	TPMLimit              int                  `json:"tpm_limit,omitempty"`
	UpdatedAt             string               `json:"updated_at,omitempty"`
}

// CreateTeam creates a new team in the Litellm service
func (l *LitellmClient) CreateTeam(ctx context.Context, req *TeamRequest) (TeamResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal team request payload")
		return TeamResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/team/new", body)
	if err != nil {
		log.Error(err, "Failed to create team in Litellm")
		return TeamResponse{}, err
	}

	var createTeamResponse TeamResponse
	if err := json.Unmarshal(response, &createTeamResponse); err != nil {
		log.Error(err, "Failed to unmarshal create team response from Litellm")
		return TeamResponse{}, err
	}

	return createTeamResponse, nil
}

// DeleteTeam deletes a team from the Litellm service
func (l *LitellmClient) DeleteTeam(ctx context.Context, teamID string) error {
	log := log.FromContext(ctx)

	body := []byte(`{"team_ids": ["` + teamID + `"]}`)

	if _, err := l.makeRequest(ctx, "POST", "/team/delete", body); err != nil {
		log.Error(err, "Failed to delete team in Litellm")
		return err
	}

	return nil
}

func (l *LitellmClient) CheckTeamExists(ctx context.Context, teamAlias string) (bool, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/v2/team/list?team_alias="+teamAlias, nil)
	if err != nil {
		log.Error(err, "Failed to check if Team exists")
		return false, err
	}

	var response struct {
		Teams []struct {
			TeamID    string `json:"team_id"`
			TeamAlias string `json:"team_alias"`
		} `json:"teams"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return false, err
	}

	// Check if any team exists with the given alias
	// Since team aliases are unique, we only need to check the first team if any exists
	return len(response.Teams) > 0 && response.Teams[0].TeamAlias == teamAlias, nil
}
