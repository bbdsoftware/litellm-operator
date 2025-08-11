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
	"errors"
	"fmt"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	litellm.LitellmTeam
	connectionHandler *common.ConnectionHandler
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

	// Initialize connection handler if not already done
	if r.connectionHandler == nil {
		r.connectionHandler = common.NewConnectionHandler(r.Client)
	}

	// Get connection details
	connectionDetails, err := r.connectionHandler.GetConnectionDetailsFromAuthRef(ctx, team.Spec.ConnectionRef, team.Namespace)
	if err != nil {
		log.Error(err, "Failed to get connection details")
		if _, updateErr := r.updateConditions(ctx, team, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ConnectionError",
			Message:            err.Error(),
		}); updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	// Configure the LiteLLM client with connection details only if not already set (for testing)
	if r.LitellmTeam == nil {
		r.LitellmTeam = common.ConfigureLitellmClient(connectionDetails)
	}

	// If the Team is being deleted, delete the team from litellm
	if team.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(team, util.FinalizerName) {
			log.Info("Deleting Team: " + team.Status.TeamAlias + " from litellm")
			return r.deleteTeam(ctx, team)
		}
		return ctrl.Result{}, nil
	}

	teamID, err := r.GetTeamID(ctx, team.Spec.TeamAlias)
	if err != nil {
		log.Error(err, "Failed to check if Team exists")
		if _, updateErr := r.updateConditions(ctx, team, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "UnableToCheckTeamExists",
			Message:            err.Error(),
		}); updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	// Team does not exist, create it
	if teamID == "" {
		log.Info("Creating Team: " + team.Spec.TeamAlias + " in litellm")
		return r.createTeam(ctx, team)
	}

	// If the TeamID is not the same as the one in the CR, then the team is not managed by this CR
	if teamID != team.Status.TeamID {
		log.Info(fmt.Sprintf("Team with alias: %v already exists", team.Spec.TeamAlias))
		return r.updateConditions(ctx, team, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "DuplicateAlias",
			Message: "Team with this alias already exists",
		})
	}

	// Sync team if required
	err = r.syncTeam(ctx, team)
	if err != nil {
		log.Error(err, "Failed to sync team")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.Team{}).
		Complete(r)
}

// updateConditions updates the Team status with the given condition
func (r *TeamReconciler) updateConditions(ctx context.Context, team *authv1alpha1.Team, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if meta.SetStatusCondition(&team.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, team); err != nil {
			log.Error(err, "unable to update Team status with condition")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteTeam handles the deletion of a team from the litellm service
func (r *TeamReconciler) deleteTeam(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.DeleteTeam(ctx, team.Status.TeamID); err != nil {
		return r.updateConditions(ctx, team, metav1.Condition{
			Type:               "DeleteTeam",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeleteTeamFailure",
			Message:            err.Error(),
		})
	}

	controllerutil.RemoveFinalizer(team, util.FinalizerName)
	if err := r.Update(ctx, team); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Deleted Team: " + team.Status.TeamAlias + " from litellm")
	return ctrl.Result{}, nil
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

// createTeam creates a new team for the litellm service
func (r *TeamReconciler) createTeam(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	teamRequest, err := createTeamRequest(team)
	if err != nil {
		return r.updateConditions(ctx, team, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "InvalidSpec",
			Message:            err.Error(),
		})
	}

	createTeamResponse, err := r.CreateTeam(ctx, &teamRequest)
	if err != nil {
		return r.updateConditions(ctx, team, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "LitellmError",
			Message:            err.Error(),
		})
	}

	updateTeamStatus(team, createTeamResponse)
	_, err = r.updateConditions(ctx, team, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "LitellmSuccess",
		Message: "Team created in Litellm",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(team, util.FinalizerName)
	if err := r.Update(ctx, team); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Created Team: " + team.Spec.TeamAlias + " in litellm")
	return ctrl.Result{}, nil
}

// syncTeam syncs the team with the litellm service should changes be detected
func (r *TeamReconciler) syncTeam(ctx context.Context, team *authv1alpha1.Team) error {
	log := log.FromContext(ctx)

	teamRequest, err := createTeamRequest(team)
	if err != nil {
		log.Error(err, "Failed to create team request")
		return err
	}

	// Need to set the teamID in the request
	teamRequest.TeamID = team.Status.TeamID

	teamResponse, err := r.GetTeam(ctx, team.Status.TeamID)
	if err != nil {
		log.Error(err, "Failed to get team from litellm")
		return err
	}

	if r.IsTeamUpdateNeeded(ctx, &teamResponse, &teamRequest) {
		log.Info("Updating Team: " + team.Spec.TeamAlias + " in litellm")
		updatedResponse, err := r.UpdateTeam(ctx, &teamRequest)
		if err != nil {
			log.Error(err, "Failed to update team in litellm")
			return err
		}

		updateTeamStatus(team, updatedResponse)

		if err := r.Status().Update(ctx, team); err != nil {
			log.Error(err, "Failed to update Team status")
			return err
		} else {
			log.Info("Updated Team: " + team.Spec.TeamAlias + " in litellm")
		}
	}

	return nil
}

// createTeamRequest creates a TeamRequest from a Team
func createTeamRequest(team *authv1alpha1.Team) (litellm.TeamRequest, error) {
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

// updateStatus updates the status of the k8s Team from the litellm response
func updateTeamStatus(team *authv1alpha1.Team, teamResponse litellm.TeamResponse) {
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
