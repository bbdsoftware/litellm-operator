/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Team object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	team := &authv1alpha1.Team{}
	if err := r.Get(ctx, req.NamespacedName, team); err != nil {
		// If the custom resource is not found then, it usually means that it was deleted or not created
		// In this way, we will stop the reconciliation
		if apierrors.IsNotFound(err) {
			log.Info("Team resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Team")
		return ctrl.Result{}, err
	}

	if team.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(team, finalizerName) {
			log.Info("Deleting Team: " + team.Status.TeamAlias + " from litellm")
			return r.deleteTeam(ctx, team)
		}
		return ctrl.Result{}, nil
	}

	if team.Status.Conditions == nil {
		if checkTeamExists(ctx, team) {
			log.Info("Team: " + team.Spec.TeamAlias + " already exists in litellm - skipping")

			team.Status.Conditions = append(team.Status.Conditions, metav1.Condition{
				Type:               "TeamCreated",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "TeamCreated",
				Message:            "Team already exists in litellm",
			})
			if err := r.Status().Update(ctx, team); err != nil {
				log.Error(err, "unable to update Team status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		log.Info("Creating Team: " + team.Spec.TeamAlias + " in litellm")
		return r.createTeam(ctx, team)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.Team{}).
		Complete(r)
}

// deleteTeam handles the deletion of a team from the litellm service
func (r *TeamReconciler) deleteTeam(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	url := litellmBaseURL + "/team/delete"
	body := []byte(`{"team_ids": ["` + team.Status.TeamID + `"]}`)

	_, err := makeLitellmRequest(ctx, "POST", url, body)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(team, finalizerName)
	if err := r.Update(ctx, team); err != nil {
		log.Error(err, "Failed to remove finalizer from Team")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// checkTeamExists checks if a team already exists in the litellm service by alias
func checkTeamExists(ctx context.Context, team *authv1alpha1.Team) bool {
	log := log.FromContext(ctx)
	teamExists := false

	url := litellmBaseURL + "/v2/team/list?team_alias=" + team.Spec.TeamAlias

	body, err := makeLitellmRequest(ctx, "GET", url, nil)
	if err != nil {
		log.Error(err, "Failed to check if Team exists")
		return teamExists
	}

	var response struct {
		Teams []struct {
			TeamID    string `json:"team_id"`
			TeamAlias string `json:"team_alias"`
		} `json:"teams"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to parse response body")
		return teamExists
	}

	if len(response.Teams) == 0 {
		return teamExists
	}

	if response.Teams[0].TeamAlias == team.Spec.TeamAlias {
		teamExists = true
	}

	return teamExists
}

// createTeam creates a new team for the litellm service
func (r *TeamReconciler) createTeam(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	url := litellmBaseURL + "/team/new"

	jsonData, err := buildTeamRequestPayload(team.Spec)
	if err != nil {
		log.Error(err, "Failed to build request payload")
		return ctrl.Result{}, err
	}

	body, err := makeLitellmRequest(ctx, "POST", url, jsonData)

	if err != nil {
		team.Status.Conditions = append(team.Status.Conditions, metav1.Condition{
			Type:               "TeamCreated",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "TeamCreated",
			Message:            err.Error(),
		})
		if err := r.Status().Update(ctx, team); err != nil {
			log.Error(err, "unable to update Team status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Parse the response to get key information
	var response struct {
		CreatedAt             string                            `json:"created_at"`
		UpdatedAt             string                            `json:"updated_at"`
		TeamID                string                            `json:"team_id"`
		TeamAlias             string                            `json:"team_alias"`
		OrganizationID        string                            `json:"organization_id"`
		MembersWithRoles      []authv1alpha1.TeamMemberWithRole `json:"members_with_roles"`
		TeamMemberPermissions []string                          `json:"team_member_permissions"`
		TPMLimit              string                            `json:"tpm_limit"`
		RPMLimit              string                            `json:"rpm_limit"`
		MaxBudget             string                            `json:"max_budget"`
		BudgetDuration        string                            `json:"budget_duration"`
		BudgetResetAt         string                            `json:"budget_reset_at"`
		Models                []string                          `json:"models"`
		Metadata              map[string]string                 `json:"metadata"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to parse response body")
		return ctrl.Result{}, err
	}

	team.Status.CreatedAt = response.CreatedAt
	team.Status.UpdatedAt = response.UpdatedAt
	team.Status.TeamID = response.TeamID
	team.Status.TeamAlias = response.TeamAlias
	team.Status.OrganizationID = response.OrganizationID
	team.Status.MembersWithRole = response.MembersWithRoles
	team.Status.TeamMemberPermissions = response.TeamMemberPermissions
	team.Status.TPMLimit = response.TPMLimit
	team.Status.RPMLimit = response.RPMLimit
	team.Status.MaxBudget = response.MaxBudget
	team.Status.BudgetDuration = response.BudgetDuration
	team.Status.BudgetResetAt = response.BudgetResetAt
	team.Status.Models = response.Models
	team.Status.Metadata = response.Metadata

	team.Status.Conditions = append(team.Status.Conditions, metav1.Condition{
		Type:               "TeamCreated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "TeamCreated",
		Message:            "Team created in litellm",
	})

	if err := r.Status().Update(ctx, team); err != nil {
		log.Error(err, "unable to update Team status")
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(team, finalizerName)
	if err := r.Update(ctx, team); err != nil {
		log.Error(err, "Failed to add finalizer to Team")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildTeamRequestPayload builds the request payload for the createTeam function
func buildTeamRequestPayload(spec authv1alpha1.TeamSpec) ([]byte, error) {
	payload := map[string]interface{}{
		"team_alias": spec.TeamAlias,
	}
	if spec.TeamID != "" {
		payload["team_id"] = spec.TeamID
	}
	if spec.OrganizationID != "" {
		payload["organization_id"] = spec.OrganizationID
	}
	if spec.MembersWithRole != nil {
		membersWithRoles := []map[string]string{}
		for _, member := range spec.MembersWithRole {
			memberMap := make(map[string]string)
			if member.UserID != "" {
				memberMap["user_id"] = member.UserID
			}
			if member.UserEmail != "" {
				memberMap["user_email"] = member.UserEmail
			}
			if member.Role != "" {
				memberMap["role"] = member.Role
			}
			membersWithRoles = append(membersWithRoles, memberMap)
		}
		payload["members_with_roles"] = membersWithRoles
	}
	if spec.TeamMemberPermissions != nil {
		payload["team_member_permissions"] = spec.TeamMemberPermissions
	}
	if spec.TPMLimit != "" {
		payload["tpm_limit"] = spec.TPMLimit
	}
	if spec.RPMLimit != "" {
		payload["rpm_limit"] = spec.RPMLimit
	}
	if spec.MaxBudget != "" {
		payload["max_budget"] = spec.MaxBudget
	}
	if spec.BudgetDuration != "" {
		payload["budget_duration"] = spec.BudgetDuration
	}
	if spec.Models != nil {
		payload["models"] = spec.Models
	}
	if spec.Tags != nil {
		payload["tags"] = spec.Tags
	}
	operatorMetadata := map[string]string{
		"managed_by": "litellm-operator",
	}
	if spec.Metadata != nil {
		for k, v := range spec.Metadata {
			operatorMetadata[k] = v
		}
	}
	payload["metadata"] = operatorMetadata

	return json.Marshal(payload)
}
