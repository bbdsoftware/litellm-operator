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

// TeamMemberAssociationReconciler reconciles a TeamMemberAssociation object
type TeamMemberAssociationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	litellm.LitellmTeamMemberAssociation
	connectionHandler *common.ConnectionHandler
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=teammemberassociations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TeamMemberAssociation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *TeamMemberAssociationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	teamMemberAssociation := &authv1alpha1.TeamMemberAssociation{}
	if err := r.Get(ctx, req.NamespacedName, teamMemberAssociation); err != nil {
		// If the custom resource is not found then, it usually means that it was deleted or not created
		// In this way, we will stop the reconciliation
		if apierrors.IsNotFound(err) {
			log.Info("TeamMemberAssociation resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get TeamMemberAssociation")
		return ctrl.Result{}, err
	}

	// Initialize connection handler if not already done
	if r.connectionHandler == nil {
		r.connectionHandler = common.NewConnectionHandler(r.Client)
	}

	// Get connection details
	connectionDetails, err := r.connectionHandler.GetConnectionDetailsFromAuthRef(ctx, teamMemberAssociation.Spec.ConnectionRef, teamMemberAssociation.Namespace)
	if err != nil {
		log.Error(err, "Failed to get connection details")
		if _, updateErr := r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
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
	if r.LitellmTeamMemberAssociation == nil {
		r.LitellmTeamMemberAssociation = common.ConfigureLitellmClient(connectionDetails)
	}

	// If the TeamMemberAssociation is being deleted, delete the team member association from litellm
	if teamMemberAssociation.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(teamMemberAssociation, util.FinalizerName) {
			log.Info("Deleting TeamMemberAssociation: " + teamMemberAssociation.Status.TeamAlias + " from litellm")
			return r.deleteTeamMemberAssociation(ctx, teamMemberAssociation)
		}
		return ctrl.Result{}, nil
	}

	teamID, err := r.GetTeamID(ctx, teamMemberAssociation.Status.TeamAlias)
	if err != nil {
		log.Error(err, "Failed to check if Team exists")
		if _, updateErr := r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
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

	teamResponse, err := r.GetTeam(ctx, teamID)
	if err != nil {
		log.Error(err, "Failed to get Team")
		return ctrl.Result{}, err
	}

	// If the user is already in the team with the correct role, return
	if isUserCorrectlyInTeam(teamResponse, teamMemberAssociation) {
		return ctrl.Result{}, nil
	}

	result, err := r.createTeamMemberAssociation(ctx, teamMemberAssociation)
	if err != nil {
		log.Error(err, "Failed to create team member association")
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamMemberAssociationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.TeamMemberAssociation{}).
		Complete(r)
}

// updateConditions updates the TeamMemberAssociation status with the given condition
func (r *TeamMemberAssociationReconciler) updateConditions(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if meta.SetStatusCondition(&teamMemberAssociation.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, teamMemberAssociation); err != nil {
			log.Error(err, "unable to update TeamMemberAssociation status with condition")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteTeamMemberAssociation removes a User from a Team in the litellm service
func (r *TeamMemberAssociationReconciler) deleteTeamMemberAssociation(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.DeleteTeamMemberAssociation(ctx, teamMemberAssociation.Status.TeamAlias, teamMemberAssociation.Status.UserEmail); err != nil {
		return r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
			Type:               "DeleteTeamMemberAssociation",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeleteTeamMemberAssociationFailure",
			Message:            err.Error(),
		})
	}
	controllerutil.RemoveFinalizer(teamMemberAssociation, util.FinalizerName)
	if err := r.Update(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("User: " + teamMemberAssociation.Status.UserEmail + " removed from Team: " + teamMemberAssociation.Status.TeamAlias)
	return ctrl.Result{}, nil
}

// isUserCorrectlyInTeam checks if the User is already in the Team with the correct role
func isUserCorrectlyInTeam(teamResponse litellm.TeamResponse, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) bool {
	for _, member := range teamResponse.MembersWithRole {
		if member.UserEmail == teamMemberAssociation.Spec.UserEmail {
			return member.Role == teamMemberAssociation.Spec.Role
		}
	}
	return false
}

func (r *TeamMemberAssociationReconciler) createTeamMemberAssociation(ctx context.Context, teamMemberAssociation *authv1alpha1.TeamMemberAssociation) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	teamRequest, err := createTeamMemberAssociationRequest(teamMemberAssociation)
	if err != nil {
		return r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "InvalidSpec",
			Message:            err.Error(),
		})
	}

	createTeamResponse, err := r.CreateTeamMemberAssociation(ctx, &teamRequest)
	if err != nil {
		return r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "LitellmError",
			Message:            err.Error(),
		})
	}

	updateTeamMemberAssociationStatus(teamMemberAssociation, createTeamResponse)
	_, err = r.updateConditions(ctx, teamMemberAssociation, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "LitellmSuccess",
		Message: "User added to Team in Litellm",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(teamMemberAssociation, util.FinalizerName)
	if err := r.Update(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	log.Info("User: " + teamMemberAssociation.Spec.UserEmail + " added to Team: " + teamMemberAssociation.Status.TeamAlias)
	return ctrl.Result{}, nil
}

// createTeamMemberAssociationRequest creates a TeamMemberAssociationRequest from a TeamMemberAssociation
func createTeamMemberAssociationRequest(teamMemberAssociation *authv1alpha1.TeamMemberAssociation) (litellm.TeamMemberAssociationRequest, error) {
	teamMemberAssociationRequest := litellm.TeamMemberAssociationRequest{
		TeamAlias: teamMemberAssociation.Status.TeamAlias,
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

// updateTeamMemberAssociationStatus updates the TeamMemberAssociation status with the given response
func updateTeamMemberAssociationStatus(teamMemberAssociation *authv1alpha1.TeamMemberAssociation, teamMemberAssociationResponse litellm.TeamMemberAssociationResponse) {
	teamMemberAssociation.Status.TeamAlias = teamMemberAssociationResponse.TeamAlias
	teamMemberAssociation.Status.TeamID = teamMemberAssociationResponse.TeamID
	teamMemberAssociation.Status.UserEmail = teamMemberAssociationResponse.UserEmail
	teamMemberAssociation.Status.UserID = teamMemberAssociationResponse.UserID
}
