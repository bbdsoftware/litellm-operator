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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	litellm.LitellmTeam
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
		teamExists, err := r.CheckTeamExists(ctx, team.Spec.TeamAlias)
		if err != nil {
			log.Error(err, "Failed to check if Team exists")
			r.appendCondition(ctx, team, metav1.Condition{
				Type:               "CreateTeam",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "CheckTeamExistsFailure",
				Message:            err.Error(),
			})
			return ctrl.Result{}, err
		}

		if teamExists {
			log.Info("Team: " + team.Spec.TeamAlias + " already exists in litellm - skipping")

			return r.appendCondition(ctx, team, metav1.Condition{
				Type:               "CreateTeam",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "CreateTeamFailure",
				Message:            "Team already exists in litellm",
			})
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

// appendCondition appends a condition to the Team status and updates the Team
func (r *TeamReconciler) appendCondition(ctx context.Context, team *authv1alpha1.Team, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	team.Status.Conditions = append(team.Status.Conditions, condition)
	if err := r.Status().Update(ctx, team); err != nil {
		log.Error(err, "unable to update Team status with condition")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deleteTeam handles the deletion of a team from the litellm service
func (r *TeamReconciler) deleteTeam(ctx context.Context, team *authv1alpha1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.DeleteTeam(ctx, team.Status.TeamID); err != nil {
		return r.appendCondition(ctx, team, metav1.Condition{
			Type:               "DeleteTeam",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeleteTeamFailure",
			Message:            err.Error(),
		})
	}

	controllerutil.RemoveFinalizer(team, finalizerName)
	log.Info("Deleted Team: " + team.Status.TeamAlias + " from litellm")
	return ctrl.Result{}, nil
}

// convertToLitellmTeamMemberWithRole converts the TeamMemberWithRole to the Litellm TeamMemberWithRole
func convertToLitellmTeamMemberWithRole(membersWithRole []authv1alpha1.TeamMemberWithRole) []litellm.TeamMemberWithRole {
	litellmMembersWithRole := []litellm.TeamMemberWithRole{}
	for _, member := range membersWithRole {
		litellmMembersWithRole = append(litellmMembersWithRole, litellm.TeamMemberWithRole{
			UserID:    member.UserID,
			UserEmail: member.UserEmail,
			Role:      member.Role,
		})
	}
	return litellmMembersWithRole
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

	createTeamResponse, err := r.CreateTeam(ctx, &litellm.CreateTeamRequest{
		TeamAlias:             team.Spec.TeamAlias,
		TeamID:                team.Spec.TeamID,
		OrganizationID:        team.Spec.OrganizationID,
		MembersWithRole:       convertToLitellmTeamMemberWithRole(team.Spec.MembersWithRole),
		TeamMemberPermissions: team.Spec.TeamMemberPermissions,
		TPMLimit:              team.Spec.TPMLimit,
		RPMLimit:              team.Spec.RPMLimit,
		MaxBudget:             team.Spec.MaxBudget,
		BudgetDuration:        team.Spec.BudgetDuration,
		Models:                team.Spec.Models,
		Tags:                  team.Spec.Tags,
		Metadata:              ensureMetadata(team.Spec.Metadata),
	})

	if err != nil {
		return r.appendCondition(ctx, team, metav1.Condition{
			Type:               "CreateTeam",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "CreateTeamFailure",
			Message:            err.Error(),
		})
	}

	team.Status.CreatedAt = createTeamResponse.CreatedAt
	team.Status.UpdatedAt = createTeamResponse.UpdatedAt
	team.Status.TeamID = createTeamResponse.TeamID
	team.Status.TeamAlias = createTeamResponse.TeamAlias
	team.Status.OrganizationID = createTeamResponse.OrganizationID
	team.Status.MembersWithRole = convertToK8sTeamMemberWithRole(createTeamResponse.MembersWithRole)
	team.Status.TeamMemberPermissions = createTeamResponse.TeamMemberPermissions
	team.Status.TPMLimit = createTeamResponse.TPMLimit
	team.Status.RPMLimit = createTeamResponse.RPMLimit
	team.Status.MaxBudget = fmt.Sprintf("%.2f", createTeamResponse.MaxBudget)
	team.Status.BudgetDuration = createTeamResponse.BudgetDuration
	team.Status.BudgetResetAt = createTeamResponse.BudgetResetAt
	team.Status.Models = createTeamResponse.Models
	team.Status.Tags = createTeamResponse.Tags
	team.Status.Metadata = createTeamResponse.Metadata

	if _, err := r.appendCondition(ctx, team, metav1.Condition{
		Type:               "TeamCreated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "TeamCreated",
		Message:            "Team created in litellm",
	}); err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(team, finalizerName)
	log.Info("Created Team: " + team.Spec.TeamAlias + " in litellm")
	return ctrl.Result{}, nil
}
