package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmTeamMemberAssociation interface {
	CreateTeamMemberAssociation(ctx context.Context, req *TeamMemberAssociationRequest) (TeamMemberAssociationResponse, error)
	DeleteTeamMemberAssociation(ctx context.Context, teamAlias string, userEmail string) error
	GetTeam(ctx context.Context, teamID string) (TeamResponse, error)
	GetTeamID(ctx context.Context, teamAlias string) (string, error)
	GetUserID(ctx context.Context, userEmail string) (string, error)
}

type TeamMemberAssociationRequest struct {
	MaxBudgetInTeam float64 `json:"max_budget_in_team,omitempty"`
	Role            string  `json:"role,omitempty"`
	TeamAlias       string  `json:"team_alias,omitempty"`
	UserEmail       string  `json:"user_email,omitempty"`
}

type TeamMemberAssociationResponse struct {
	TeamAlias string `json:"team_alias,omitempty"`
	TeamID    string `json:"team_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// CreateTeamMemberAssociation adds a User to a Team in the Litellm service
func (l *LitellmClient) CreateTeamMemberAssociation(ctx context.Context, req *TeamMemberAssociationRequest) (TeamMemberAssociationResponse, error) {
	log := log.FromContext(ctx)

	teamID, err := l.GetTeamID(ctx, req.TeamAlias)
	if err != nil {
		log.Error(err, "Failed to get team ID")
		return TeamMemberAssociationResponse{}, err
	}

	userID, err := l.GetUserID(ctx, req.UserEmail)
	if err != nil {
		log.Error(err, "Failed to get user ID")
		return TeamMemberAssociationResponse{}, err
	}

	type addRequest struct {
		Member          []TeamMemberWithRole `json:"member,omitempty"`
		MaxBudgetInTeam float64              `json:"max_budget_in_team,omitempty"`
		TeamID          string               `json:"team_id,omitempty"`
	}

	body, err := json.Marshal(addRequest{
		Member: []TeamMemberWithRole{
			{
				UserID:    userID,
				UserEmail: req.UserEmail,
				Role:      req.Role,
			},
		},
		MaxBudgetInTeam: req.MaxBudgetInTeam,
		TeamID:          teamID,
	})
	if err != nil {
		log.Error(err, "Failed to marshal team member association request payload")
		return TeamMemberAssociationResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/team/member_add", body)
	if err != nil {
		log.Error(err, "Failed to create team member association in Litellm")
		return TeamMemberAssociationResponse{}, err
	}

	var createTeamMemberAssociationResponse TeamMemberAssociationResponse
	if err := json.Unmarshal(response, &createTeamMemberAssociationResponse); err != nil {
		log.Error(err, "Failed to unmarshal create team member association response from Litellm")
		return TeamMemberAssociationResponse{}, err
	}

	// Add user email and user id, as they don't come back from the create request
	createTeamMemberAssociationResponse.UserEmail = req.UserEmail
	createTeamMemberAssociationResponse.UserID = userID

	return createTeamMemberAssociationResponse, nil
}

// DeleteTeamMemberAssociation removes a User from a Team in the Litellm service
func (l *LitellmClient) DeleteTeamMemberAssociation(ctx context.Context, teamAlias string, userEmail string) error {
	log := log.FromContext(ctx)

	teamID, err := l.GetTeamID(ctx, teamAlias)
	if err != nil {
		log.Error(err, "Failed to get team ID")
		return err
	}

	userID, err := l.GetUserID(ctx, userEmail)
	if err != nil {
		log.Error(err, "Failed to get user ID")
		return err
	}

	type deleteRequest struct {
		TeamID    string `json:"team_id"`
		UserEmail string `json:"user_email"`
		UserID    string `json:"user_id"`
	}

	body, err := json.Marshal(deleteRequest{
		TeamID:    teamID,
		UserEmail: userEmail,
		UserID:    userID,
	})
	if err != nil {
		log.Error(err, "Failed to marshal team member association delete request payload")
		return err
	}

	if _, err := l.makeRequest(ctx, "POST", "/team/member_delete", body); err != nil {
		log.Error(err, "Failed to delete team member association in Litellm")
		return err
	}

	return nil
}
