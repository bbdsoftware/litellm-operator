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

package association

import (
	"context"
	"errors"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// TeamMemberAssociationReconciler reconciles a TeamMemberAssociation object
type TeamMemberAssociationReconciler struct {
	*base.BaseController[*authv1alpha1.TeamMemberAssociation]
	LitellmClient litellm.LitellmTeamMemberAssociation
}

// NewTeamMemberAssociationReconciler creates a new TeamMemberAssociationReconciler instance
func NewTeamMemberAssociationReconciler(client client.Client, scheme *runtime.Scheme) *TeamMemberAssociationReconciler {
	return &TeamMemberAssociationReconciler{
		BaseController: &base.BaseController[*authv1alpha1.TeamMemberAssociation]{
			Client:         client,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
		},
		LitellmClient: nil,
	}
}

type ExternalData struct {
	TeamAlias string `json:"teamAlias"`
	TeamID    string `json:"teamID"`
	UserEmail string `json:"userEmail"`
	UserID    string `json:"userID"`
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations/finalizers,verbs=update

// Reconcile implements the single-loop ensure* pattern with finalizer, conditions, and drift sync
func (r *TeamMemberAssociationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Add timeout to avoid long-running reconciliation
	ctx, cancel := r.WithTimeout(ctx)
	defer cancel()

	log := log.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	teamMemberAssociation := &authv1alpha1.TeamMemberAssociation{}
	teamMemberAssociation, err := r.FetchResource(ctx, req.NamespacedName, teamMemberAssociation)
	if err != nil {
		log.Error(err, "Failed to get TeamMemberAssociation")
		return ctrl.Result{}, err
	}
	if teamMemberAssociation == nil {
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling external team member association resource", "teamMemberAssociation", teamMemberAssociation.Name) // Add timeout to avoid long-running reconciliation
	// Phase 2: Set up connections and clients
	if err := r.ensureConnectionSetup(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to setup connections")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConnectionError)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if !teamMemberAssociation.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, teamMemberAssociation)
	}

	// Phase 4: Upsert branch - ensure finalizer
	if err := r.AddFinalizer(ctx, teamMemberAssociation, util.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	var externalData ExternalData
	// Phase 5: Ensure external resource (create/patch/repair drift)
	if res, err := r.ensureExternal(ctx, teamMemberAssociation, &externalData); res.Requeue || res.RequeueAfter > 0 || err != nil {
		return res, err
	}

	// Phase 6: Ensure in-cluster children (none for team member associations)
	// TeamMemberAssociation does not have any child resources

	// Phase 7: Mark Ready and persist ObservedGeneration
	r.SetSuccessConditions(teamMemberAssociation, "TeamMemberAssociation is in desired state")
	if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
		return ctrl.Result{}, err
	}

	// Phase 8: Periodic drift sync (external might change out of band)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// ensureConnectionSetup configures the LiteLLM client
func (r *TeamMemberAssociationReconciler) ensureConnectionSetup(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) error {
	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, teamMemberAssociation.Spec.ConnectionRef, teamMemberAssociation.Namespace)
		if err != nil {
			return err
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}
	return nil
}

// reconcileDelete handles the deletion branch with idempotent external cleanup
func (r *TeamMemberAssociationReconciler) reconcileDelete(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !r.HasFinalizer(teamMemberAssociation, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Set deleting condition and update status
	r.SetCondition(teamMemberAssociation, base.CondReady, metav1.ConditionFalse, "Deleting", "TeamMemberAssociation is being deleted")
	if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to update status during deletion")
		// Continue with deletion even if status update fails
	}

	// Idempotent external cleanup
	if teamMemberAssociation.Status.TeamAlias != "" && teamMemberAssociation.Status.UserEmail != "" {
		if err := r.LitellmClient.DeleteTeamMemberAssociation(ctx, teamMemberAssociation.Status.TeamAlias, teamMemberAssociation.Status.UserEmail); err != nil {
			log.Error(err, "Failed to delete team member association from LiteLLM")
			return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonDeleteFailed)
		}
		log.Info("Successfully deleted team member association from LiteLLM", "teamAlias", teamMemberAssociation.Status.TeamAlias, "userEmail", teamMemberAssociation.Status.UserEmail)
	}

	// Remove finalizer
	if err := r.RemoveFinalizer(ctx, teamMemberAssociation, util.FinalizerName); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonDeleteFailed)
	}

	log.Info("Successfully deleted team member association", "association", teamMemberAssociation.Name)
	return ctrl.Result{}, nil
}

// ensureExternal manages the external team member association resource (create/patch/repair drift)
func (r *TeamMemberAssociationReconciler) ensureExternal(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation, externalData *ExternalData) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Ensuring external team member association resource", "association", teamMemberAssociation.Name)

	// Set progressing condition
	r.SetProgressingConditions(teamMemberAssociation, "Reconciling team member association in LiteLLM")
	if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to update progressing status")
		// Continue despite status update failure
	}

	// Use team alias from spec, not status
	teamAlias := teamMemberAssociation.Spec.TeamAlias

	// Validate team existence
	teamID, err := r.LitellmClient.GetTeamID(ctx, teamAlias)
	if err != nil {
		log.Error(err, "Failed to validate team existence", "teamAlias", teamAlias)
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConfigError)
	}

	// Get current team state to check if user is already correctly associated
	teamResponse, err := r.LitellmClient.GetTeam(ctx, teamID)
	if err != nil {
		log.Error(err, "Failed to get team state")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonLitellmError)
	}

	// Check if user is already correctly in team
	if r.isUserCorrectlyInTeam(teamResponse, teamMemberAssociation) {
		log.V(1).Info("User is already correctly associated with team", "userEmail", teamMemberAssociation.Spec.UserEmail, "teamAlias", teamAlias)
		// Update status with current state
		externalData.TeamAlias = teamAlias
		externalData.TeamID = teamID
		externalData.UserEmail = teamMemberAssociation.Spec.UserEmail
		// Find user ID from team members
		for _, member := range teamResponse.MembersWithRole {
			if member.UserEmail == teamMemberAssociation.Spec.UserEmail {
				externalData.UserID = member.UserID
				break
			}
		}
		r.updateTeamMemberAssociationStatus(teamMemberAssociation, externalData)
		if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
			log.Error(err, "Failed to update status")
			return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonReconcileError)
		}
		return ctrl.Result{}, nil
	}

	// Create or update the association
	log.Info("Creating team member association in LiteLLM", "userEmail", teamMemberAssociation.Spec.UserEmail, "teamAlias", teamAlias)

	associationRequest, err := r.convertToTeamMemberAssociationRequest(teamMemberAssociation)
	if err != nil {
		log.Error(err, "Failed to create team member association request")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonInvalidSpec)
	}

	createResponse, err := r.LitellmClient.CreateTeamMemberAssociation(ctx, &associationRequest)
	if err != nil {
		log.Error(err, "Failed to create team member association in LiteLLM")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonLitellmError)
	}

	externalData.TeamAlias = createResponse.TeamAlias
	externalData.TeamID = createResponse.TeamID
	externalData.UserEmail = createResponse.UserEmail
	externalData.UserID = createResponse.UserID

	r.updateTeamMemberAssociationStatus(teamMemberAssociation, externalData)
	if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to update status after creation")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonReconcileError)
	}
	log.Info("Successfully created team member association in LiteLLM", "userEmail", teamMemberAssociation.Spec.UserEmail, "teamAlias", teamAlias)
	return ctrl.Result{}, nil
}

// isUserCorrectlyInTeam checks if the User is already in the Team with the correct role
func (r *TeamMemberAssociationReconciler) isUserCorrectlyInTeam(teamResponse litellm.TeamResponse, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) bool {
	for _, member := range teamResponse.MembersWithRole {
		if member.UserEmail == teamMemberAssociation.Spec.UserEmail {
			return member.Role == teamMemberAssociation.Spec.Role
		}
	}
	return false
}

// convertToTeamMemberAssociationRequest creates a TeamMemberAssociationRequest from a TeamMemberAssociation (isolated for testing)
func (r *TeamMemberAssociationReconciler) convertToTeamMemberAssociationRequest(teamMemberAssociation *authv1alpha1.TeamMemberAssociation) (litellm.TeamMemberAssociationRequest, error) {
	teamMemberAssociationRequest := litellm.TeamMemberAssociationRequest{
		TeamAlias: teamMemberAssociation.Spec.TeamAlias,
		UserEmail: teamMemberAssociation.Spec.UserEmail,
		Role:      teamMemberAssociation.Spec.Role,
	}

	if teamMemberAssociation.Spec.MaxBudgetInTeam != "" {
		maxBudget, err := strconv.ParseFloat(teamMemberAssociation.Spec.MaxBudgetInTeam, 64)
		if err != nil {
			return litellm.TeamMemberAssociationRequest{}, errors.New("maxBudget: " + err.Error())
		}
		teamMemberAssociationRequest.MaxBudgetInTeam = maxBudget
	}

	return teamMemberAssociationRequest, nil
}

// updateTeamMemberAssociationStatus updates the status of the k8s TeamMemberAssociation from the external data
func (r *TeamMemberAssociationReconciler) updateTeamMemberAssociationStatus(teamMemberAssociation *authv1alpha1.TeamMemberAssociation, externalData *ExternalData) {
	teamMemberAssociation.Status.TeamAlias = externalData.TeamAlias
	teamMemberAssociation.Status.TeamID = externalData.TeamID
	teamMemberAssociation.Status.UserEmail = externalData.UserEmail
	teamMemberAssociation.Status.UserID = externalData.UserID
}

func (r *TeamMemberAssociationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.TeamMemberAssociation{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-teammemberassociation").
		Complete(r)
}
