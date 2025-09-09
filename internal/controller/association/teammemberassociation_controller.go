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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

	//Phase 1.1 : validate Team and User readiness
	team := &authv1alpha1.Team{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: teamMemberAssociation.Namespace, Name: teamMemberAssociation.Spec.TeamRef.Name}, team); err != nil {
		log.Error(err, "Failed to get referenced Team", "teamRef", teamMemberAssociation.Spec.TeamRef.Name)
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConfigError)
	}
	if !IsConditionTrue(team.Status.Conditions, base.CondReady) {
		err := errors.New("referenced Team is not ready")
		log.Error(err, "Referenced Team is not ready", "teamRef", teamMemberAssociation.Spec.TeamRef.Name)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	user := &authv1alpha1.User{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: teamMemberAssociation.Namespace, Name: teamMemberAssociation.Spec.UserRef.Name}, user); err != nil {
		log.Error(err, "Failed to get referenced User", "userRef", teamMemberAssociation.Spec.UserRef.Name)
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConfigError)
	}
	if !IsConditionTrue(user.Status.Conditions, base.CondReady) {
		err := errors.New("referenced User is not ready")
		log.Error(err, "Referenced User is not ready", "userRef", teamMemberAssociation.Spec.UserRef.Name)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}
	teamMemberAssociation.Status.TeamExists = true
	teamMemberAssociation.Status.UserExists = true
	teamMemberAssociation.Status.AssociationIsValid = true

	if err := r.PatchStatus(ctx, teamMemberAssociation); err != nil {
		log.Error(err, "Failed to update status after validating references")
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonReconcileError)
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

func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
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

	// Validate team existence
	team := &authv1alpha1.Team{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: teamMemberAssociation.Namespace, Name: teamMemberAssociation.Spec.TeamRef.Name}, team); err != nil {
		log.Error(err, "Failed to get referenced Team", "teamRef", teamMemberAssociation.Spec.TeamRef.Name)
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConfigError)
	}
	teamAlias := team.Spec.TeamAlias
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
	user := &authv1alpha1.User{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: teamMemberAssociation.Namespace, Name: teamMemberAssociation.Spec.UserRef.Name}, user); err != nil {
		log.Error(err, "Failed to get referenced User", "userRef", teamMemberAssociation.Spec.UserRef.Name)
		return r.HandleErrorRetryable(ctx, teamMemberAssociation, err, base.ReasonConfigError)
	}
	userEmail := user.Spec.UserEmail

	if r.isUserCorrectlyInTeam(teamResponse, teamMemberAssociation, userEmail) {
		log.V(1).Info("User is already correctly associated with team", "userEmail", userEmail, "teamAlias", teamAlias)
		// Update status with current state
		externalData.TeamAlias = teamAlias
		externalData.TeamID = teamID
		externalData.UserEmail = userEmail
		// Find user ID from team members
		for _, member := range teamResponse.MembersWithRole {
			if member.UserEmail == userEmail {
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
	log.Info("Creating team member association in LiteLLM", "userEmail", userEmail, "teamAlias", teamAlias)

	associationRequest, err := r.convertToTeamMemberAssociationRequest(teamMemberAssociation, userEmail, teamAlias)
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
	log.Info("Successfully created team member association in LiteLLM", "userEmail", userEmail, "teamAlias", teamAlias)
	return ctrl.Result{}, nil
}

// isUserCorrectlyInTeam checks if the User is already in the Team with the correct role
func (r *TeamMemberAssociationReconciler) isUserCorrectlyInTeam(teamResponse litellm.TeamResponse, teamMemberAssociation *authv1alpha1.TeamMemberAssociation, userEmail string) bool {
	for _, member := range teamResponse.MembersWithRole {
		if member.UserEmail == userEmail {
			return member.Role == teamMemberAssociation.Spec.Role
		}
	}
	return false
}

// convertToTeamMemberAssociationRequest creates a TeamMemberAssociationRequest from a TeamMemberAssociation (isolated for testing)
func (r *TeamMemberAssociationReconciler) convertToTeamMemberAssociationRequest(teamMemberAssociation *authv1alpha1.TeamMemberAssociation, userEmail string, teamAlias string) (litellm.TeamMemberAssociationRequest, error) {
	teamMemberAssociationRequest := litellm.TeamMemberAssociationRequest{
		TeamAlias: teamAlias,
		UserEmail: userEmail,
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

// mapUserToAssociations finds all TeamAssociations that reference a specific User
func (r *TeamMemberAssociationReconciler) mapUserToAssociations(obj client.Object) []reconcile.Request {
	user := obj.(*authv1alpha1.User)
	var requests []reconcile.Request

	// List all TeamMemberAssociations in the same namespace
	assocList := &authv1alpha1.TeamMemberAssociationList{}
	if err := r.List(context.Background(), assocList, client.InNamespace(user.Namespace)); err != nil {
		return []reconcile.Request{}
	}

	// Find associations that reference this user
	for _, assoc := range assocList.Items {
		// Match by userRef if present (name required, namespace optional)
		if assoc.Spec.UserRef.Name != "" {
			if assoc.Spec.UserRef.Name == user.Name {
				// If the association omits a namespace, treat it as same-namespace as the association
				if assoc.Spec.UserRef.Namespace == "" || assoc.Spec.UserRef.Namespace == user.Namespace {
					requests = append(requests, reconcile.Request{
						NamespacedName: client.ObjectKey{Name: assoc.Name, Namespace: assoc.Namespace},
					})
					continue
				}
			}
		}
	}
	return requests
}

func (r *TeamMemberAssociationReconciler) mapTeamToAssociations(obj client.Object) []reconcile.Request {
	team := obj.(*authv1alpha1.Team)
	var requests []reconcile.Request

	// List all TeamMemberAssociations in the same namespace
	assocList := &authv1alpha1.TeamMemberAssociationList{}
	if err := r.List(context.Background(), assocList, client.InNamespace(team.Namespace)); err != nil {
		return []reconcile.Request{}
	}

	// Find associations that reference this team
	for _, assoc := range assocList.Items {
		// Match by teamRef if present
		if assoc.Spec.TeamRef.Name != "" {
			if assoc.Spec.TeamRef.Name == team.Name {
				if assoc.Spec.TeamRef.Namespace == "" || assoc.Spec.TeamRef.Namespace == team.Namespace {
					requests = append(requests, reconcile.Request{
						NamespacedName: client.ObjectKey{Name: assoc.Name, Namespace: assoc.Namespace},
					})
					continue
				}
			}
		}
	}
	return requests
}

// updateTeamMemberAssociationStatus updates the status of the k8s TeamMemberAssociation from the external data
func (r *TeamMemberAssociationReconciler) updateTeamMemberAssociationStatus(teamMemberAssociation *authv1alpha1.TeamMemberAssociation, externalData *ExternalData) {
	teamMemberAssociation.Status.TeamAlias = externalData.TeamAlias
	teamMemberAssociation.Status.TeamID = externalData.TeamID
	teamMemberAssociation.Status.UserEmail = externalData.UserEmail
	teamMemberAssociation.Status.UserID = externalData.UserID
}

func (r *TeamMemberAssociationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create typed EventHandler for User objects
	userHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		return r.mapUserToAssociations(obj)
	})

	// Create typed EventHandler for Team objects
	teamHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		return r.mapTeamToAssociations(obj)
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.TeamMemberAssociation{}).
		Watches(&authv1alpha1.User{}, userHandler).
		Watches(&authv1alpha1.Team{}, teamHandler).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-teammemberassociation").
		Complete(r)
}
