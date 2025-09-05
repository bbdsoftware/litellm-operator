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

package team

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	litellm "github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	*base.BaseController[*authv1alpha1.Team]
	LitellmClient litellm.LitellmTeam
}

// NewTeamReconciler creates a new TeamReconciler instance
func NewTeamReconciler(client client.Client, scheme *runtime.Scheme) *TeamReconciler {
	return &TeamReconciler{
		BaseController: &base.BaseController[*authv1alpha1.Team]{
			Client:         client,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
		},
		LitellmClient: nil,
	}
}

type ExternalData struct {
	TeamID    string `json:"teamID"`
	TeamAlias string `json:"teamAlias"`
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teams/finalizers,verbs=update

// Reconcile implements the single-loop ensure* pattern with finalizer, conditions, and drift sync
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Add timeout to avoid long-running reconciliation
	ctx, cancel := r.WithTimeout(ctx)
	defer cancel()

	log := log.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	team := &authv1alpha1.Team{}
	team, err := r.FetchResource(ctx, req.NamespacedName, team)
	if err != nil {
		log.Error(err, "Failed to get Team")
		return ctrl.Result{}, err
	}
	if team == nil {
		return ctrl.Result{}, nil
	}

	// Phase 2: Set up connections and clients
	if err := r.ensureConnectionSetup(ctx, team); err != nil {
		log.Error(err, "Failed to setup connections")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonConnectionError)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if !team.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, team)
	}

	// Phase 4: Upsert branch - ensure finalizer
	if err := r.AddFinalizer(ctx, team, util.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	var externalData ExternalData
	// Phase 5: Ensure external resource (create/patch/repair drift)
	if res, err := r.ensureExternal(ctx, team, &externalData); res.Requeue || res.RequeueAfter > 0 || err != nil {
		return res, err
	}

	// Phase 6: Ensure in-cluster children (owned -> GC on delete)
	if err := r.ensureChildren(ctx, team, &externalData); err != nil {
		return r.HandleCommonErrors(ctx, team, err)
	}

	// Phase 7: Mark Ready and persist ObservedGeneration
	r.SetSuccessConditions(team, "Team is in desired state")
	team.Status.ObservedGeneration = team.GetGeneration()
	if err := r.PatchStatus(ctx, team); err != nil {
		return ctrl.Result{}, err
	}

	// Phase 8: Periodic drift sync (external might change out of band)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// ensureConnectionSetup configures the LiteLLM client
func (r *TeamReconciler) ensureConnectionSetup(ctx context.Context, team *authv1alpha1.Team) error {
	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, team.Spec.ConnectionRef, team.Namespace)
		if err != nil {
			return err
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}

	return nil
}

// reconcileDelete handles the deletion branch with idempotent external cleanup
func (r *TeamReconciler) reconcileDelete(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !r.HasFinalizer(team, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Set deleting condition and update status
	r.SetCondition(team, base.CondReady, metav1.ConditionFalse, "Deleting", "Team is being deleted")
	if err := r.PatchStatus(ctx, team); err != nil {
		log.Error(err, "Failed to update status during deletion")
		// Continue with deletion even if status update fails
	}

	// Idempotent external cleanup
	if team.Status.TeamID != "" {
		if err := r.LitellmClient.DeleteTeam(ctx, team.Status.TeamID); err != nil {
			log.Error(err, "Failed to delete team from LiteLLM")
			return r.HandleErrorRetryable(ctx, team, err, base.ReasonDeleteFailed)
		}
		log.Info("Successfully deleted team from LiteLLM", "teamID", team.Status.TeamID)
	}

	// Remove finalizer
	if err := r.RemoveFinalizer(ctx, team, util.FinalizerName); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonDeleteFailed)
	}

	log.Info("Successfully deleted team", "team", team.Name)
	return ctrl.Result{}, nil
}

// ensureExternal manages the external team resource (create/patch/repair drift)
func (r *TeamReconciler) ensureExternal(ctx context.Context, team *authv1alpha1.Team, externalData *ExternalData) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Ensuring external team resource", "team", team.Name)

	// Set progressing condition
	r.SetProgressingConditions(team, "Reconciling team in LiteLLM")
	if err := r.PatchStatus(ctx, team); err != nil {
		log.Error(err, "Failed to update progressing status")
		// Continue despite status update failure
	}

	teamRequest, err := r.convertToTeamRequest(team)
	if err != nil {
		log.Error(err, "Failed to create team request")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonInvalidSpec)
	}

	// Check if team exists by alias
	existingTeamID, err := r.LitellmClient.GetTeamID(ctx, team.Spec.TeamAlias)
	if err != nil {
		log.Error(err, "Failed to check if team exists")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonLitellmError)
	}

	// If team exists but doesn't match our managed team ID, it's a conflict
	if existingTeamID != "" && team.Status.TeamID != "" && existingTeamID != team.Status.TeamID {
		err := fmt.Errorf("team with alias %s already exists with different ID (existing: %s, ours: %s)",
			team.Spec.TeamAlias, existingTeamID, team.Status.TeamID)
		log.Error(err, "Team alias conflict")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonConfigError)
	}

	// Create if no external ID exists or if no team found by alias
	if team.Status.TeamID == "" || existingTeamID == "" {
		log.Info("Creating new team in LiteLLM", "teamAlias", team.Spec.TeamAlias)
		createResponse, err := r.LitellmClient.CreateTeam(ctx, &teamRequest)
		if err != nil {
			log.Error(err, "Failed to create team in LiteLLM")
			return r.HandleErrorRetryable(ctx, team, err, base.ReasonLitellmError)
		}

		externalData.TeamID = createResponse.TeamID
		externalData.TeamAlias = createResponse.TeamAlias

		r.updateTeamStatus(team, createResponse)
		if err := r.PatchStatus(ctx, team); err != nil {
			log.Error(err, "Failed to update status after creation")
			return r.HandleErrorRetryable(ctx, team, err, base.ReasonReconcileError)
		}
		log.Info("Successfully created team in LiteLLM", "teamID", createResponse.TeamID)
		return ctrl.Result{}, nil
	}

	// Team exists, check for drift and repair if needed
	log.V(1).Info("Checking for drift", "teamID", team.Status.TeamID)
	observedTeam, err := r.LitellmClient.GetTeam(ctx, team.Status.TeamID)
	if err != nil {
		log.Error(err, "Failed to get team from LiteLLM")
		return r.HandleErrorRetryable(ctx, team, err, base.ReasonLitellmError)
	}

	// Set the teamID in the request for update
	teamRequest.TeamID = team.Status.TeamID

	updateNeeded := r.LitellmClient.IsTeamUpdateNeeded(ctx, &observedTeam, &teamRequest)
	if updateNeeded {
		log.Info("Repairing drift in LiteLLM", "teamAlias", team.Spec.TeamAlias)
		updateResponse, err := r.LitellmClient.UpdateTeam(ctx, &teamRequest)
		if err != nil {
			log.Error(err, "Failed to update team in LiteLLM")
			return r.HandleErrorRetryable(ctx, team, err, base.ReasonLitellmError)
		}

		externalData.TeamID = updateResponse.TeamID
		externalData.TeamAlias = updateResponse.TeamAlias

		r.updateTeamStatus(team, updateResponse)
		if err := r.PatchStatus(ctx, team); err != nil {
			log.Error(err, "Failed to update status after update")
			return r.HandleErrorRetryable(ctx, team, err, base.ReasonReconcileError)
		}
		log.Info("Successfully repaired drift in LiteLLM", "teamID", team.Status.TeamID)
	} else {
		log.V(1).Info("Team is up to date in LiteLLM", "teamID", team.Status.TeamID)
	}

	return ctrl.Result{}, nil
}

// ensureChildren manages in-cluster child resources (teams don't have children currently)
func (r *TeamReconciler) ensureChildren(ctx context.Context, team *authv1alpha1.Team, externalData *ExternalData) error {
	// Teams don't currently have child resources, but this follows the pattern
	return nil
}

// convertToTeamRequest creates a TeamRequest from a Team (isolated for testing)
func (r *TeamReconciler) convertToTeamRequest(team *authv1alpha1.Team) (litellm.TeamRequest, error) {
	teamRequest := litellm.TeamRequest{
		Blocked:               team.Spec.Blocked,
		BudgetDuration:        team.Spec.BudgetDuration,
		Guardrails:            team.Spec.Guardrails,
		Metadata:              util.EnsureMetadata(team.Spec.Metadata),
		ModelAliases:          team.Spec.ModelAliases,
		Models:                team.Spec.Models,
		OrganizationID:        team.Spec.OrganizationID,
		RPMLimit:              team.Spec.RPMLimit,
		Tags:                  team.Spec.Tags,
		TeamAlias:             team.Spec.TeamAlias,
		TeamID:                team.Spec.TeamID,
		TeamMemberPermissions: team.Spec.TeamMemberPermissions,
		TPMLimit:              team.Spec.TPMLimit,
	}

	if team.Spec.MaxBudget != "" {
		maxBudget, err := strconv.ParseFloat(team.Spec.MaxBudget, 64)
		if err != nil {
			return litellm.TeamRequest{}, errors.New("maxBudget: " + err.Error())
		}
		teamRequest.MaxBudget = maxBudget
	}

	return teamRequest, nil
}

// updateTeamStatus updates the status of the k8s Team from the litellm response
func (r *TeamReconciler) updateTeamStatus(team *authv1alpha1.Team, teamResponse litellm.TeamResponse) {
	team.Status.Blocked = teamResponse.Blocked
	team.Status.BudgetDuration = teamResponse.BudgetDuration
	team.Status.BudgetResetAt = teamResponse.BudgetResetAt
	team.Status.CreatedAt = teamResponse.CreatedAt
	team.Status.LiteLLMModelTable = teamResponse.LiteLLMModelTable
	team.Status.MaxBudget = fmt.Sprintf("%.2f", teamResponse.MaxBudget)
	team.Status.MaxParallelRequests = teamResponse.MaxParallelRequests
	team.Status.MembersWithRole = convertToK8sTeamMemberWithRole(teamResponse.MembersWithRole)
	team.Status.ModelID = teamResponse.ModelID
	team.Status.Models = teamResponse.Models
	team.Status.OrganizationID = teamResponse.OrganizationID
	team.Status.RPMLimit = teamResponse.RPMLimit
	team.Status.Spend = fmt.Sprintf("%.2f", teamResponse.Spend)
	team.Status.Tags = teamResponse.Tags
	team.Status.TeamAlias = teamResponse.TeamAlias
	team.Status.TeamID = teamResponse.TeamID
	team.Status.TeamMemberPermissions = teamResponse.TeamMemberPermissions
	team.Status.TPMLimit = teamResponse.TPMLimit
	team.Status.UpdatedAt = teamResponse.UpdatedAt
}

// convertToK8sTeamMemberWithRole converts the Litellm TeamMemberWithRole to TeamMemberWithRole
func convertToK8sTeamMemberWithRole(membersWithRole []litellm.TeamMemberWithRole) []authv1alpha1.TeamMemberWithRole {
	k8sMembersWithRole := []authv1alpha1.TeamMemberWithRole{}
	for _, member := range membersWithRole {
		k8sMembersWithRole = append(k8sMembersWithRole, authv1alpha1.TeamMemberWithRole{
			UserID:    member.UserID,
			UserEmail: member.UserEmail,
			Role:      member.Role,
		})
	}
	return k8sMembersWithRole
}

func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.Team{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-team").
		Complete(r)
}
