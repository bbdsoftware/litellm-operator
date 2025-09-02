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

package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	litellm "github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	LitellmClient         litellm.LitellmUser
	litellmResourceNaming *util.LitellmResourceNaming
	OverrideLiteLLMURL    string
}

// Constants moved to controller.go

const (
	CondReady       = "Ready"       // Overall health indicator
	CondProgressing = "Progressing" // Actively reconciling / rolling out
	CondDegraded    = "Degraded"
	CondFailed      = "Failed"
)

// ---------- Machine-friendly reasons (stable enums; messages can be free text)
const (
	ReasonReconciling         = "Reconciling"
	ReasonConfigError         = "ConfigError"
	ReasonChildCreateOrUpdate = "ChildResourcesUpdating"
	ReasonDependencyNotReady  = "DependencyNotReady"
	ReasonReconcileError      = "ReconcileError"
	ReasonReady               = "Ready"
	ReasonDeleted             = "Deleted"
	ReasonConnectionError     = "ConnectionError"
	ReasonDeleteFailed        = "DeleteFailed"
	ReasonCreateFailed        = "CreateFailed"
	ReasonConversionFailed    = "ConversionFailed"
	ReasonUpdateFailed        = "UpdateFailed"
	ReasonTeamCheckFailed     = "TeamCheckFailed"
	ReasonDuplicateEmail      = "DuplicateEmail"
	ReasonInvalidSpec         = "InvalidSpec"
	ReasonLitellmError        = "LitellmError"
	ReasonLitellmSuccess      = "LitellmSuccess"
)

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/finalizers,verbs=update

// fetchUser retrieves the User resource and handles not found errors gracefully
func (r *UserReconciler) fetchUser(ctx context.Context, namespacedName client.ObjectKey) (*authv1alpha1.User, error) {
	log := log.FromContext(ctx)
	user := &authv1alpha1.User{}
	err := r.Get(ctx, namespacedName, user)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("User resource not found. Ignoring since object must be deleted")
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	user, err := r.fetchUser(ctx, req.NamespacedName)
	if err != nil {
		log.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}
	if user == nil {
		// Resource was deleted, stop reconciliation
		return ctrl.Result{}, nil
	}

	// If a LitellmClient was injected (tests), reuse it; otherwise create one from connection details
	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, user.Spec.ConnectionRef, user.Namespace)
		if err != nil {
			r.setCond(user, CondDegraded, metav1.ConditionTrue, ReasonConnectionError, err.Error())
			r.setCond(user, CondReady, metav1.ConditionFalse, ReasonConnectionError, err.Error())
			if err := r.patchStatus(ctx, user); err != nil {
				log.Error(err, "Failed to update status after connection error")
			}
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&user.Spec.ConnectionRef)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if user.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, user, req)
	}

	// Phase 4: Handle creation/update (normal reconciliation)
	return r.handleCreateOrUpdateUser(ctx, user, req)

}

func shouldSyncUser(user *authv1alpha1.User) bool {
	return user.Status.UserID != "" && user.Status.UserID != user.Spec.UserID
}

func (r *UserReconciler) userExistsInLitellm(ctx context.Context, user *authv1alpha1.User) (bool, error) {
	userID, err := r.LitellmClient.GetUserID(ctx, user.Spec.UserEmail)
	if err != nil {
		return false, err
	}
	return userID != "", nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-user").
		Complete(r)
}

// handleDeletion manages the user deletion process with proper finalizer handling
func (r *UserReconciler) handleDeletion(ctx context.Context, user *authv1alpha1.User, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(user, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Deleting User from LiteLLM", "user", user.Status.UserAlias, "userID", user.Status.UserID)

	// Delete the user from LiteLLM
	if err := r.deleteUserFromLitellm(ctx, user, r.LitellmClient); err != nil {
		log.Error(err, "Failed to delete user from LiteLLM")
		r.setCond(user, CondDegraded, metav1.ConditionTrue, ReasonDeleteFailed, err.Error())
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonDeleteFailed, err.Error())
		if updateErr := r.patchStatus(ctx, user, req); updateErr != nil {
			log.Error(updateErr, "Failed to update status after delete user from LiteLLM error")
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	controllerutil.RemoveFinalizer(user, util.FinalizerName)
	if err := r.Update(ctx, user); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted User from LiteLLM and removed finalizer", "user", user.Status.UserAlias)
	return ctrl.Result{}, nil
}

func (r *UserReconciler) deleteUserFromLitellm(ctx context.Context, user *authv1alpha1.User, litellmClient litellm.LitellmUser) error {
	log := log.FromContext(ctx)

	if err := litellmClient.DeleteUser(ctx, user.Status.UserID); err != nil {
		return err
	}
	log.Info("Deleted User: " + user.Status.UserAlias + " from litellm")
	return nil
}

// handleCreateOrUpdateUser manages the complete user lifecycle (creation and updates)
func (r *UserReconciler) handleCreateOrUpdateUser(ctx context.Context, user *authv1alpha1.User, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(user, util.FinalizerName) {
		controllerutil.AddFinalizer(user, util.FinalizerName)
		if err := r.Update(ctx, user); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Finalizer added", "user", user.Name)
	}

	// Check that the teamIDs exist before attempting to create the user
	for _, teamID := range user.Spec.Teams {
		_, err := r.LitellmClient.GetTeam(ctx, teamID)
		if err != nil {
			r.setCond(user, CondDegraded, metav1.ConditionTrue, ReasonTeamCheckFailed, err.Error())
			r.setCond(user, CondReady, metav1.ConditionFalse, ReasonTeamCheckFailed, err.Error())
			if err := r.patchStatus(ctx, user); err != nil {
				log.Error(err, "Failed to update status after team check error")
			}
			return ctrl.Result{}, nil
		}
	}

	userExists, err := r.userExistsInLitellm(ctx, user)
	if err != nil {
		log.Error(err, "Failed to check if User exists")
		r.setCond(user, CondDegraded, metav1.ConditionTrue, ReasonConnectionError, err.Error())
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonConnectionError, err.Error())
		if err := r.patchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update status after GetUserID error")
		}
		return ctrl.Result{}, err
	}

	// User does not exist, create it
	if userID == "" {
		log.Info("Creating User: " + user.Spec.UserAlias + " in litellm")
		return r.createUser(ctx, user)
	}

	// If the UserID is not the same as the one in the CR, then the user is not managed by this CR
	if userID != user.Status.UserID {
		errorMessage := fmt.Sprintf("User with email %s already exists but is not managed by this resource (existing ID: %s, expected ID: %s)", user.Spec.UserEmail, userID, user.Status.UserID)
		log.Info(errorMessage)
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonDuplicateEmail, errorMessage)
		r.setCond(user, CondFailed, metav1.ConditionTrue, ReasonDuplicateEmail, errorMessage)
		if err := r.patchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update status after duplicate email detection")
		}
		return ctrl.Result{}, nil
	}

	// Sync user if required
	err = r.syncUser(ctx, user)
	if err != nil {
		log.Error(err, "Failed to sync user")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		Complete(r)
}

// deleteUser handles the deletion of a user from the litellm service
func (r *UserReconciler) deleteUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.LitellmClient.DeleteUser(ctx, user.Status.UserID); err != nil {
		r.setCond(user, CondDegraded, metav1.ConditionTrue, ReasonDeleteFailed, err.Error())
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonDeleteFailed, err.Error())
		if err := r.patchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(user, util.FinalizerName)
	if err := r.Update(ctx, user); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Deleted User: " + user.Status.UserAlias + " from litellm")
	return ctrl.Result{}, nil
}

// createUser creates a new user for the litellm service
func (r *UserReconciler) createUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	userRequest, err := createUserRequest(user)
	if err != nil {
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonInvalidSpec, err.Error())
		if err := r.patchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	userResponse, err := r.LitellmClient.CreateUser(ctx, &userRequest)
	if err != nil {
		r.setCond(user, CondReady, metav1.ConditionFalse, ReasonLitellmError, err.Error())
		if errPatch := r.patchStatus(ctx, user); errPatch != nil {
			log.Error(errPatch, "Failed to update conditions")
		}

		log.Info("Created User: " + user.Spec.UserAlias + " in litellm")
		return ctrl.Result{}, nil
	}

	secretName := r.litellmResourceNaming.GenerateSecretName(userResponse.UserAlias)

	updateUserStatus(user, userResponse, secretName)

	// Add finalizer first
	controllerutil.AddFinalizer(user, util.FinalizerName)
	if errAddFinalizer := r.Update(ctx, user); errAddFinalizer != nil {
		log.Error(errAddFinalizer, "Failed to add finalizer")
		return ctrl.Result{}, errAddFinalizer
	}

	// Create the secret
	if err := r.createSecret(ctx, user, secretName, userResponse.Key); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
	}

	// Update status conditions
	r.setCond(user, CondReady, metav1.ConditionTrue, ReasonLitellmSuccess, "User created in Litellm")
	if errPatch := r.patchStatus(ctx, user); errPatch != nil {
		log.Error(errPatch, "Failed to update conditions")
	}

	log.Info("Created User: " + user.Spec.UserAlias + " in litellm")
	return ctrl.Result{}, nil
}

// createSecret stores the secret key in a Kubernetes Secret that is owned by the User
func (r *UserReconciler) createSecret(ctx context.Context, user *authv1alpha1.User, secretName string, key string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: user.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "auth.litellm.ai/v1alpha1",
					Kind:       "User",
					Name:       user.Name,
					UID:        user.UID,
				},
			},
		},
		Data: map[string][]byte{
			"key": []byte(key),
		},
	}

	return r.Create(ctx, secret)
}

// updateUserInLitellm updates the user with the litellm service should changes be detected
func (r *UserReconciler) handleUpdateUser(ctx context.Context, user *authv1alpha1.User) error {
	log := log.FromContext(ctx)

	userRequest, err := createUserRequest(user)
	if err != nil {
		return err
	}

	userResponse, err := r.LitellmClient.GetUser(ctx, user.Status.UserID)
	if err != nil {
		log.Error(err, "Failed to get user from litellm")
		return err
	}

	userUpdateNeeded, err := r.LitellmClient.IsUserUpdateNeeded(ctx, &userResponse, &userRequest)
	if err != nil {
		log.Error(err, "Failed to check if user needs to be updated")
		return err
	}
	if userUpdateNeeded.NeedsUpdate {
		log.Info("Updating User: "+user.Spec.UserAlias+" in litellm", "Fields changed", userUpdateNeeded.ChangedFields)
		updatedResponse, err := r.LitellmClient.UpdateUser(ctx, &userRequest)
		if err != nil {
			log.Error(err, "Failed to update user in litellm")
			return err
		}

		updateUserStatus(user, updatedResponse, user.Status.KeySecretRef)

		if err := r.Status().Update(ctx, user); err != nil {
			log.Error(err, "Failed to update User status")
			return err
		} else {
			log.Info("Updated User: " + user.Spec.UserAlias + " in litellm")
		}
	}

	return nil
}

// createUserRequest creates a UserRequest from a User
func createUserRequest(user *authv1alpha1.User) (litellm.UserRequest, error) {
	userRequest := litellm.UserRequest{
		Aliases:              user.Spec.Aliases,
		AllowedCacheControls: user.Spec.AllowedCacheControls,
		AutoCreateKey:        user.Spec.AutoCreateKey,
		Blocked:              user.Spec.Blocked,
		BudgetDuration:       user.Spec.BudgetDuration,
		Duration:             user.Spec.Duration,
		Guardrails:           user.Spec.Guardrails,
		KeyAlias:             user.Spec.KeyAlias,
		MaxParallelRequests:  user.Spec.MaxParallelRequests,
		Metadata:             util.EnsureMetadata(user.Spec.Metadata),
		ModelMaxBudget:       user.Spec.ModelMaxBudget,
		ModelRPMLimit:        user.Spec.ModelRPMLimit,
		ModelTPMLimit:        user.Spec.ModelTPMLimit,
		Models:               user.Spec.Models,
		Permissions:          user.Spec.Permissions,
		RPMLimit:             user.Spec.RPMLimit,
		SendInviteEmail:      user.Spec.SendInviteEmail,
		SSOUserID:            user.Spec.SSOUserID,
		Teams:                user.Spec.Teams,
		TPMLimit:             user.Spec.TPMLimit,
		UserAlias:            user.Spec.UserAlias,
		UserEmail:            user.Spec.UserEmail,
		UserID:               user.Spec.UserID,
		UserRole:             user.Spec.UserRole,
	}

	if user.Spec.MaxBudget != "" {
		maxBudget, err := strconv.ParseFloat(user.Spec.MaxBudget, 64)
		if err != nil {
			return litellm.UserRequest{}, errors.New("maxBudget: " + err.Error())
		}
		userRequest.MaxBudget = maxBudget
	}
	if user.Spec.SoftBudget != "" {
		softBudget, err := strconv.ParseFloat(user.Spec.SoftBudget, 64)
		if err != nil {
			return litellm.UserRequest{}, errors.New("softBudget: " + err.Error())
		}
		userRequest.SoftBudget = softBudget
	}

	return userRequest, nil
}

// updateUserStatus updates the status of the k8s User from the litellm response
func updateUserStatus(user *authv1alpha1.User, userResponse litellm.UserResponse, secretKeyName string) {
	user.Status.Aliases = userResponse.Aliases
	user.Status.AllowedCacheControls = userResponse.AllowedCacheControls
	user.Status.AllowedRoutes = userResponse.AllowedRoutes
	user.Status.Blocked = userResponse.Blocked
	user.Status.BudgetDuration = userResponse.BudgetDuration
	user.Status.BudgetID = userResponse.BudgetID
	user.Status.CreatedAt = userResponse.CreatedAt
	user.Status.CreatedBy = userResponse.CreatedBy
	user.Status.Duration = userResponse.Duration
	user.Status.EnforcedParams = userResponse.EnforcedParams
	user.Status.Expires = userResponse.Expires
	user.Status.Guardrails = userResponse.Guardrails
	user.Status.KeyAlias = userResponse.KeyAlias
	user.Status.KeyName = userResponse.KeyName
	user.Status.KeySecretRef = secretKeyName
	user.Status.LiteLLMBudgetTable = userResponse.LiteLLMBudgetTable
	user.Status.MaxBudget = fmt.Sprintf("%.2f", userResponse.MaxBudget)
	user.Status.MaxParallelRequests = userResponse.MaxParallelRequests
	// These don't actually come back here, they are injected into the metadata field which complicates things, so skip for now
	// user.Status.ModelMaxBudget = userResponse.ModelMaxBudget
	// user.Status.ModelRPMLimit = userResponse.ModelRPMLimit
	// user.Status.ModelTPMLimit = userResponse.ModelTPMLimit
	user.Status.Models = userResponse.Models
	user.Status.Permissions = userResponse.Permissions
	user.Status.RPMLimit = userResponse.RPMLimit
	user.Status.Spend = fmt.Sprintf("%.2f", userResponse.Spend)
	user.Status.Tags = userResponse.Tags
	user.Status.Teams = userResponse.Teams
	user.Status.TPMLimit = userResponse.TPMLimit
	user.Status.UpdatedAt = userResponse.UpdatedAt
	user.Status.UpdatedBy = userResponse.UpdatedBy
	user.Status.UserAlias = userResponse.UserAlias
	user.Status.UserEmail = userResponse.UserEmail
	user.Status.UserID = userResponse.UserID
	user.Status.UserRole = userResponse.UserRole
}

// ============================================================================
// Utility Functions
// ============================================================================

// patchStatus updates the status subresource
func (r *UserReconciler) patchStatus(ctx context.Context, cr *authv1alpha1.User) error {
	return r.Status().Update(ctx, cr)
}

func (r *UserReconciler) setCond(cr *authv1alpha1.User, typeCond string, status metav1.ConditionStatus, reason, msg string) {
	newC := metav1.Condition{
		Type:               typeCond,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: cr.GetGeneration(),
	}
	meta.SetStatusCondition(&cr.Status.Conditions, newC)
}
