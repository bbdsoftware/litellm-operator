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

	corev1 "k8s.io/api/core/v1"
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
	litellm "github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	litellm.LitellmUser
	connectionHandler     *common.ConnectionHandler
	litellmResourceNaming *util.LitellmResourceNaming
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

	user := &authv1alpha1.User{}
	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		// If the custom resource is not found then, it usually means that it was deleted or not created
		// In this way, we will stop the reconciliation
		if apierrors.IsNotFound(err) {
			log.Info("User resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}

	// Initialize connection handler if not already done
	if r.connectionHandler == nil {
		r.connectionHandler = common.NewConnectionHandler(r.Client)
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&user.Spec.ConnectionRef)
	}

	// Get connection details
	connectionDetails, err := r.connectionHandler.GetConnectionDetailsFromAuthRef(ctx, user.Spec.ConnectionRef, user.Namespace)
	if err != nil {
		log.Error(err, "Failed to get connection details")
		if _, updateErr := r.updateConditions(ctx, user, metav1.Condition{
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
	if r.LitellmUser == nil {
		r.LitellmUser = common.ConfigureLitellmClient(connectionDetails)
	}

	// If the User is being deleted, delete the user from litellm
	if user.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(user, util.FinalizerName) {
			log.Info("Deleting User: " + user.Status.UserAlias + " from litellm")
			return r.deleteUser(ctx, user)
		}
		return ctrl.Result{}, nil
	}

	// Check that the teamIDs exist before attempting to create the user
	for _, teamID := range user.Spec.Teams {
		_, err := r.GetTeam(ctx, teamID)
		if err != nil {
			if _, updateErr := r.updateConditions(ctx, user, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "TeamCheckFailed",
				Message: err.Error(),
			}); updateErr != nil {
				log.Error(updateErr, "Failed to update conditions")
			}
			return ctrl.Result{}, nil
		}
	}

	userID, err := r.GetUserID(ctx, user.Spec.UserEmail)
	if err != nil {
		log.Error(err, "Failed to check if User exists")
		if _, updateErr := r.updateConditions(ctx, user, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "UnableToCheckUserExists",
			Message: err.Error(),
		}); updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
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
		log.Info(fmt.Sprintf("User with email: %v already exists", user.Spec.UserEmail))
		return r.updateConditions(ctx, user, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "DuplicateEmail",
			Message: "User with this email already exists",
		})
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

// updateConditions updates the User status with the given condition
func (r *UserReconciler) updateConditions(ctx context.Context, user *authv1alpha1.User, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if meta.SetStatusCondition(&user.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, user); err != nil {
			log.Error(err, "unable to update User status with condition")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteUser handles the deletion of a user from the litellm service
func (r *UserReconciler) deleteUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.DeleteUser(ctx, user.Status.UserID); err != nil {
		return r.updateConditions(ctx, user, metav1.Condition{
			Type:    "DeleteUser",
			Status:  metav1.ConditionFalse,
			Reason:  "DeleteUserFailure",
			Message: err.Error(),
		})
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
		if _, err := r.updateConditions(ctx, user, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	userResponse, err := r.CreateUser(ctx, &userRequest)
	if err != nil {
		return r.updateConditions(ctx, user, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "LitellmError",
			Message: err.Error(),
		})
	}

	secretName := r.litellmResourceNaming.GenerateSecretName(userResponse.UserAlias)

	updateUserStatus(user, userResponse, secretName)
	_, err = r.updateConditions(ctx, user, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "LitellmSuccess",
		Message: "User created in Litellm",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(user, util.FinalizerName)
	if err := r.Update(ctx, user); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	if err := r.createSecret(ctx, user, secretName, userResponse.Key); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
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

// syncUser syncs the user with the litellm service should changes be detected
func (r *UserReconciler) syncUser(ctx context.Context, user *authv1alpha1.User) error {
	log := log.FromContext(ctx)

	userRequest, err := createUserRequest(user)
	if err != nil {
		log.Error(err, "Failed to create user request")
		return err
	}

	// Need to set the userID in the request, else it will generate a new one
	userRequest.UserID = user.Status.UserID

	userResponse, err := r.GetUser(ctx, user.Status.UserID)
	if err != nil {
		log.Error(err, "Failed to get user from litellm")
		return err
	}

	if r.IsUserUpdateNeeded(ctx, &userResponse, &userRequest) {
		log.Info("Updating User: " + user.Spec.UserAlias + " in litellm")
		updatedResponse, err := r.UpdateUser(ctx, &userRequest)
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
