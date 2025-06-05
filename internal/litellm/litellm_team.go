package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TeamMemberWithRole struct {
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	Role      string `json:"role,omitempty"`
}

type CreateTeamRequest struct {
	TeamAlias             string               `json:"team_alias,omitempty"`
	TeamID                string               `json:"team_id,omitempty"`
	OrganizationID        string               `json:"organization_id,omitempty"`
	MembersWithRole       []TeamMemberWithRole `json:"members_with_role,omitempty"`
	TeamMemberPermissions []string             `json:"team_member_permissions,omitempty"`
	TPMLimit              string               `json:"tpm_limit,omitempty"`
	RPMLimit              string               `json:"rpm_limit,omitempty"`
	MaxBudget             string               `json:"max_budget,omitempty"`
	BudgetDuration        string               `json:"budget_duration,omitempty"`
	Models                []string             `json:"models,omitempty"`
	Tags                  []string             `json:"tags,omitempty"`
	Metadata              map[string]string    `json:"metadata,omitempty"`
}

type CreateTeamResponse struct {
	CreatedAt             string               `json:"created_at"`
	UpdatedAt             string               `json:"updated_at"`
	TeamID                string               `json:"team_id"`
	TeamAlias             string               `json:"team_alias"`
	OrganizationID        string               `json:"organization_id"`
	MembersWithRole       []TeamMemberWithRole `json:"members_with_role"`
	TeamMemberPermissions []string             `json:"team_member_permissions"`
	TPMLimit              string               `json:"tpm_limit"`
	RPMLimit              string               `json:"rpm_limit"`
	MaxBudget             float64              `json:"max_budget"`
	BudgetDuration        string               `json:"budget_duration"`
	BudgetResetAt         string               `json:"budget_reset_at"`
	Models                []string             `json:"models"`
	Tags                  []string             `json:"tags"`
	Metadata              map[string]string    `json:"metadata"`
}

// CreateTeam creates a new team in the Litellm service
func (l *LitellmClient) CreateTeam(ctx context.Context, req *CreateTeamRequest) (CreateTeamResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal team request payload")
		return CreateTeamResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/team/new", body)
	if err != nil {
		log.Error(err, "Failed to create team in Litellm")
		return CreateTeamResponse{}, err
	}

	var createTeamResponse CreateTeamResponse
	if err := json.Unmarshal(response, &createTeamResponse); err != nil {
		log.Error(err, "Failed to unmarshal create team response from Litellm")
		return CreateTeamResponse{}, err
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
