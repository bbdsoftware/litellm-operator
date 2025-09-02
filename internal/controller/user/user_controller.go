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
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	litellm "github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	*base.BaseController[*authv1alpha1.User]
	LitellmClient         litellm.LitellmUser
	litellmResourceNaming *util.LitellmResourceNaming
}

// NewUserReconciler creates a new UserReconciler instance
func NewUserReconciler(client client.Client, scheme *runtime.Scheme) *UserReconciler {
	return &UserReconciler{
		BaseController: &base.BaseController[*authv1alpha1.User]{
			Client: client,
			Scheme: scheme,
		},
		LitellmClient:         nil,
		litellmResourceNaming: nil,
	}
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/finalizers,verbs=update

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
	user := &authv1alpha1.User{}
	user, err := r.FetchResource(ctx, req.NamespacedName, user)
	if err != nil {
		log.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}
	if user == nil {
		// Resource was deleted, stop reconciliation
		return ctrl.Result{}, nil
	}

	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, user.Spec.ConnectionRef, user.Namespace)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, user, err, base.ReasonConnectionError)
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&user.Spec.ConnectionRef)
	}


	// Normal path: ensure finalizer
	if cr.DeletionTimestamp.IsZero() {
		if err := r.AddFinalizer(ctx, &cr, finalizerUser); err != nil {
			return ctrl.Result{}, err
		}
		return r.reconcileUser(ctx, &cr)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if user.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, user)
	}

	// Phase 4: Handle creation/update (normal reconciliation)
	return r.handleCreateOrUpdateUser(ctx, user)

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
func (r *UserReconciler) handleDeletion(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	return r.HandleDeletionWithFinalizer(ctx, user, util.FinalizerName, func(ctx context.Context, user *authv1alpha1.User) error {
		return r.deleteUserFromLitellm(ctx, user, r.LitellmClient)
	})
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
func (r *UserReconciler) reconcileUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var observedUser litellm.UserResponse
	if desiredUser, err := createUserRequest(user); err != nil {
		log.Error(err, "Failed to create user request")
		return ctrl.Result{}, err
	}


	if user.Status.UserID == "" {
		
		observedUser, err := r.LitellmClient.CreateUser(ctx, &desiredUser)
		if err != nil {
			log.Error(err, "Failed to create user in litellm")
			return ctrl.Result{}, err
		}
		updateUserStatus(user, userResponse, desiredUser.KeyAlias)
		

	}else{
		observedUser, err := r.LitellmClient.GetUser(ctx, user.Status.UserID)
		if err != nil {
			log.Error(err, "Failed to get user from litellm")
			return ctrl.Result{}, err
		}
	}



	observed : = 
	// Ensure finalizer is present
	if err := r.EnsureFinalizer(ctx, user, util.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	// Check that the teamIDs exist before attempting to create the user
	for _, teamID := range user.Spec.Teams {
		_, err := r.LitellmClient.GetTeam(ctx, teamID)
		if err != nil {
			return r.HandleErrorRetryable(ctx, user, err, "TeamCheckFailed", "TeamCheckFailed")
		}
	}

	userExists, err := r.userExistsInLitellm(ctx, user)
	if err != nil {
		log.Error(err, "Failed to check if User exists")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonConnectionError, "ConnectionError")
	}

	// Create the user if it doesn't exist
	if !userExists {
		log.Info("Creating User: " + user.Spec.UserAlias + " in litellm")
		return r.handleCreateUser(ctx, user)
	}

	if userExists {
		log.Info("Updating User: " + user.Spec.UserAlias + " in litellm")
		return r.handleUpdateUser(ctx, user)
	}

	return ctrl.Result{}, nil
}

// createUser creates a new user for the litellm service
func (r *UserReconciler) handleCreateUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	userRequest, err := createUserRequest(user)
	if err != nil {
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonInvalidSpec)
	}

	userResponse, err := r.LitellmClient.CreateUser(ctx, &userRequest)
	if err != nil {
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonLitellmError)
	}

	secretName := r.litellmResourceNaming.GenerateSecretName(userResponse.UserAlias)
	if err := r.createSecret(ctx, user, secretName, userResponse.Key); err != nil {
		log.Error(err, "Failed to create secret")
		return r.HandleErrorRetryable(ctx, user, err, "SecretCreateFailed")
	}

	updateUserStatus(user, userResponse, secretName)

	log.Info("Created User: " + user.Spec.UserAlias + " in litellm")
	return r.HandleSuccess(ctx, user, "User created in LiteLLM")
}

// updateUserInLitellm updates the user with the litellm service should changes be detected
func (r *UserReconciler) handleUpdateUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	userRequest, err := createUserRequest(user)
	if err != nil {
		return ctrl.Result{}, err
	}

	userResponse, err := r.LitellmClient.GetUser(ctx, user.Status.UserID)
	if err != nil {
		log.Error(err, "Failed to get user from litellm")
		return ctrl.Result{}, err
	}

	userUpdateNeeded, err := r.LitellmClient.IsUserUpdateNeeded(ctx, &userResponse, &userRequest)
	if err != nil {
		log.Error(err, "Failed to check if user needs to be updated")
		return ctrl.Result{}, err
	}

	if userUpdateNeeded.NeedsUpdate {
		log.Info("Updating User: "+user.Spec.UserAlias+" in litellm", "Fields changed", userUpdateNeeded.ChangedFields)
		updatedResponse, err := r.LitellmClient.UpdateUser(ctx, &userRequest)
		if err != nil {
			log.Error(err, "Failed to update user in litellm")
			return ctrl.Result{}, err
		}

		updateUserStatus(user, updatedResponse, user.Status.KeySecretRef)

		log.Info("Updated User: " + user.Spec.UserAlias + " in litellm")
		return r.HandleSuccess(ctx, user, "User updated in LiteLLM")
	}

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
