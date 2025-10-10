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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
			Client:         client,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
			ControllerName: "user",
		},
		LitellmClient:         nil,
		litellmResourceNaming: nil,
	}
}

type ExternalData struct {
	UserID    string `json:"userID"`
	UserEmail string `json:"userEmail"`
	UserRole  string `json:"userRole"`
	Key       string `json:"key"`
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=users/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements the single-loop ensure* pattern with finalizer, conditions, and drift sync
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Add timeout to avoid long-running reconciliation
	ctx, cancel := r.WithTimeout(ctx)
	defer cancel()

	// Instrument the reconcile loop
	r.InstrumentReconcileLoop()
	timer := r.InstrumentReconcileLatency()
	defer timer.ObserveDuration()

	log := log.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	user := &authv1alpha1.User{}
	user, err := r.FetchResource(ctx, req.NamespacedName, user)
	if err != nil {
		log.Error(err, "Failed to get User")
		r.InstrumentReconcileError()
		return ctrl.Result{}, err
	}
	if user == nil {
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling external user resource", "user", user.Name) // Add timeout to avoid long-running reconciliation
	// Phase 2: Set up connections and clients
	if err := r.ensureConnectionSetup(ctx, user); err != nil {
		log.Error(err, "Failed to setup connections")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonConnectionError)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if !user.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, user)
	}

	// Phase 4: Upsert branch - ensure finalizer
	if err := r.AddFinalizer(ctx, user, util.FinalizerName); err != nil {
		r.InstrumentReconcileError()
		return ctrl.Result{}, err
	}

	var externalData ExternalData
	// Phase 5: Ensure external resource (create/patch/repair drift)
	if res, err := r.ensureExternal(ctx, user, &externalData); res.Requeue || res.RequeueAfter > 0 || err != nil {
		r.InstrumentReconcileError()
		return res, err
	}

	// Phase 6: Ensure in-cluster children (owned -> GC on delete)
	if err := r.ensureChildren(ctx, user, &externalData); err != nil {
		return r.HandleCommonErrors(ctx, user, err)
	}

	// Phase 7: Mark Ready and persist ObservedGeneration
	r.SetSuccessConditions(user, "User is in desired state")
	user.Status.ObservedGeneration = user.GetGeneration()
	if err := r.PatchStatus(ctx, user); err != nil {
		return ctrl.Result{}, err
	}

	// Phase 8: Periodic drift sync (external might change out of band)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// ensureConnectionSetup configures the LiteLLM client and resource naming
func (r *UserReconciler) ensureConnectionSetup(ctx context.Context, user *authv1alpha1.User) error {
	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, user.Spec.ConnectionRef, user.Namespace)
		if err != nil {
			return err
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&user.Spec.ConnectionRef)
	}

	return nil
}

// reconcileDelete handles the deletion branch with idempotent external cleanup
func (r *UserReconciler) reconcileDelete(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !r.HasFinalizer(user, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Set deleting condition and update status
	r.SetCondition(user, base.CondReady, metav1.ConditionFalse, "Deleting", "User is being deleted")
	if err := r.PatchStatus(ctx, user); err != nil {
		log.Error(err, "Failed to update status during deletion")
		// Continue with deletion even if status update fails
	}

	// Idempotent external cleanup
	if user.Status.UserID != "" {
		if err := r.LitellmClient.DeleteUser(ctx, user.Status.UserID); err != nil {
			log.Error(err, "Failed to delete user from LiteLLM")
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonDeleteFailed)
		}
		log.Info("Successfully deleted user from LiteLLM", "userID", user.Status.UserID)
	}

	// Remove finalizer
	if err := r.RemoveFinalizer(ctx, user, util.FinalizerName); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonDeleteFailed)
	}

	log.Info("Successfully deleted user", "user", user.Name)
	return ctrl.Result{}, nil
}

// ensureExternal manages the external user resource (create/patch/repair drift)
func (r *UserReconciler) ensureExternal(ctx context.Context, user *authv1alpha1.User, externalData *ExternalData) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Ensuring external user resource", "user", user.Name)

	// Set progressing condition
	r.SetProgressingConditions(user, "Reconciling user in LiteLLM")
	if err := r.PatchStatus(ctx, user); err != nil {
		log.Error(err, "Failed to update progressing status")
		// Continue despite status update failure
	}

	// Validate that all referenced teams exist
	for _, teamID := range user.Spec.Teams {
		_, err := r.LitellmClient.GetTeam(ctx, teamID)
		if err != nil {
			log.Error(err, "Failed to validate team existence", "teamID", teamID)
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonConfigError)
		}
	}

	desiredUser, err := r.convertToUserRequest(user)
	if err != nil {
		log.Error(err, "Failed to create user request")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonInvalidSpec)
	}

	// Create if no external ID exists
	if user.Status.UserID == "" {
		log.Info("Creating new user in LiteLLM", "userAlias", user.Spec.UserAlias)
		createResponse, err := r.LitellmClient.CreateUser(ctx, &desiredUser)
		if err != nil {
			log.Error(err, "Failed to create user in LiteLLM")
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonLitellmError)
		}

		externalData.UserID = createResponse.UserID
		externalData.UserEmail = createResponse.UserEmail
		externalData.UserRole = createResponse.UserRole
		externalData.Key = createResponse.Key

		r.updateUserStatus(user, createResponse, desiredUser.KeyAlias)
		if err := r.PatchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update status after creation")
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonReconcileError)
		}
		log.Info("Successfully created user in LiteLLM", "userID", createResponse.UserID)
		return ctrl.Result{}, nil
	}

	// User exists, check for drift and repair if needed
	log.V(1).Info("Checking for drift", "userID", user.Status.UserID)
	observedUser, err := r.LitellmClient.GetUser(ctx, user.Status.UserID)
	if err != nil {
		log.Error(err, "Failed to get user from LiteLLM")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonLitellmError)
	}

	updateNeeded, err := r.LitellmClient.IsUserUpdateNeeded(ctx, &observedUser, &desiredUser)
	if err != nil {
		log.Error(err, "Failed to check if user needs update")
		return r.HandleErrorRetryable(ctx, user, err, base.ReasonLitellmError)
	}

	if updateNeeded.NeedsUpdate {
		log.Info("Repairing drift in LiteLLM", "userAlias", user.Spec.UserAlias, "changedFields", updateNeeded.ChangedFields)
		updateResponse, err := r.LitellmClient.UpdateUser(ctx, &desiredUser)
		if err != nil {
			log.Error(err, "Failed to update user in LiteLLM")
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonLitellmError)
		}

		externalData.UserID = updateResponse.UserID
		externalData.UserEmail = updateResponse.UserEmail
		externalData.UserRole = updateResponse.UserRole
		externalData.Key = updateResponse.Key
		r.updateUserStatus(user, updateResponse, user.Status.KeySecretRef)
		if err := r.PatchStatus(ctx, user); err != nil {
			log.Error(err, "Failed to update status after update")
			return r.HandleErrorRetryable(ctx, user, err, base.ReasonReconcileError)
		}
		log.Info("Successfully repaired drift in LiteLLM", "userID", user.Status.UserID)
	} else {
		log.V(1).Info("User is up to date in LiteLLM", "userID", user.Status.UserID)
	}

	return ctrl.Result{}, nil
}

// ensureChildren manages in-cluster child resources using CreateOrUpdate pattern
func (r *UserReconciler) ensureChildren(ctx context.Context, user *authv1alpha1.User, externalData *ExternalData) error {
	if user.Status.KeySecretRef == "" {
		return nil // No secret to create
	}

	secretName := r.litellmResourceNaming.GenerateSecretName(user.Spec.KeyAlias)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: user.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		// Set controller reference for garbage collection
		if err := controllerutil.SetControllerReference(user, secret, r.Scheme); err != nil {
			return err
		}

		// Update secret data
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data["key"] = []byte(externalData.Key)

		return nil
	})

	return err
}

// convertToUserRequest creates a UserRequest from a User (isolated for testing)
func (r *UserReconciler) convertToUserRequest(user *authv1alpha1.User) (litellm.UserRequest, error) {
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
func (r *UserReconciler) updateUserStatus(user *authv1alpha1.User, userResponse litellm.UserResponse, secretKeyName string) {
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

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-user").
		Complete(r)
}
