package litellm

import (
	"context"
	"encoding/json"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmTeam interface {
	CreateTeam(ctx context.Context, req *TeamRequest) (TeamResponse, error)
	DeleteTeam(ctx context.Context, teamID string) error
	GetTeam(ctx context.Context, teamID string) (TeamResponse, error)
	GetTeamID(ctx context.Context, teamAlias string) (string, error)
	IsTeamUpdateNeeded(ctx context.Context, team *TeamResponse, req *TeamRequest) bool
	UpdateTeam(ctx context.Context, req *TeamRequest) (TeamResponse, error)
}

type TeamMemberWithRole struct {
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	Role      string `json:"role,omitempty"`
}

type TeamRequest struct {
	Admins                []string          `json:"admins,omitempty"`
	Blocked               bool              `json:"blocked,omitempty"`
	BudgetDuration        string            `json:"budget_duration,omitempty"`
	Guardrails            []string          `json:"guardrails,omitempty"`
	MaxBudget             float64           `json:"max_budget,omitempty"`
	Members               []string          `json:"members,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
	ModelAliases          map[string]string `json:"model_aliases,omitempty"`
	Models                []string          `json:"models,omitempty"`
	OrganizationID        string            `json:"organization_id,omitempty"`
	RPMLimit              int               `json:"rpm_limit,omitempty"`
	Tags                  []string          `json:"tags,omitempty"`
	TeamAlias             string            `json:"team_alias,omitempty"`
	TeamID                string            `json:"team_id,omitempty"`
	TeamMemberPermissions []string          `json:"team_member_permissions,omitempty"`
	TPMLimit              int               `json:"tpm_limit,omitempty"`
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

// UpdateTeam updates a team in the Litellm service
func (l *LitellmClient) UpdateTeam(ctx context.Context, req *TeamRequest) (TeamResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal team request payload")
		return TeamResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/team/update", body)
	if err != nil {
		log.Error(err, "Failed to update team in Litellm")
		return TeamResponse{}, err
	}

	var updateTeamResponse struct {
		Team TeamResponse `json:"data"`
	}

	if err := json.Unmarshal(response, &updateTeamResponse); err != nil {
		log.Error(err, "Failed to unmarshal update team response from Litellm")
		return TeamResponse{}, err
	}

	return updateTeamResponse.Team, nil
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

// GetTeamID gets the ID of a team from the Litellm service, returns empty string if team alias not found
func (l *LitellmClient) GetTeamID(ctx context.Context, teamAlias string) (string, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/v2/team/list?team_alias="+teamAlias, nil)
	if err != nil {
		log.Error(err, "Failed to list teams")
		return "", err
	}

	var response struct {
		Teams []struct {
			TeamID    string `json:"team_id"`
			TeamAlias string `json:"team_alias"`
		} `json:"teams"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return "", err
	}

	// Check if any team exists with the given alias
	// Since team aliases are unique, we only need to check the first team if any exists
	if len(response.Teams) > 0 {
		return response.Teams[0].TeamID, nil
	}
	return "", nil
}

// GetTeam gets a team from the Litellm service
func (l *LitellmClient) GetTeam(ctx context.Context, teamID string) (TeamResponse, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/team/info?team_id="+teamID, nil)
	if err != nil {
		log.Error(err, "Failed to get team with ID: "+teamID)
		return TeamResponse{}, err
	}

	var response struct {
		Team TeamResponse `json:"team_info"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal team response from Litellm")
		return TeamResponse{}, err
	}

	return response.Team, nil
}

// IsTeamUpdateNeeded checks if a team needs to be updated in the Litellm service
func (l *LitellmClient) IsTeamUpdateNeeded(ctx context.Context, team *TeamResponse, req *TeamRequest) bool {
	log := log.FromContext(ctx)

	if !cmp.Equal(team.Admins, req.Admins, cmpopts.EquateEmpty()) {
		log.Info("Admins changed")
		return true
	}

	if team.Blocked != req.Blocked {
		log.Info("Blocked changed")
		return true
	}

	if team.BudgetDuration != req.BudgetDuration {
		log.Info("BudgetDuration changed")
		return true
	}

	if team.MaxBudget != req.MaxBudget {
		log.Info("MaxBudget changed")
		return true
	}

	if !cmp.Equal(team.Members, req.Members, cmpopts.EquateEmpty()) {
		log.Info("Members changed")
		return true
	}

	if !cmp.Equal(team.OrganizationID, req.OrganizationID, cmpopts.EquateEmpty()) {
		log.Info("OrganizationID changed")
		return true
	}

	if !cmp.Equal(team.RPMLimit, req.RPMLimit, cmpopts.EquateEmpty()) {
		log.Info("RPMLimit changed")
		return true
	}

	if !cmp.Equal(team.TeamAlias, req.TeamAlias, cmpopts.EquateEmpty()) {
		log.Info("TeamAlias changed")
		return true
	}

	if team.TeamID != req.TeamID {
		log.Info("TeamID changed")
		return true
	}

	if !cmp.Equal(team.TeamMemberPermissions, req.TeamMemberPermissions, cmpopts.EquateEmpty()) {
		log.Info("TeamMemberPermissions changed")
		return true
	}

	if team.TPMLimit != req.TPMLimit {
		log.Info("TPMLimit changed")
		return true
	}

	return false
}
