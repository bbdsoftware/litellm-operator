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

package virtualkey

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// VirtualKeyReconciler reconciles a VirtualKey object
type VirtualKeyReconciler struct {
	*base.BaseController[*authv1alpha1.VirtualKey]
	LitellmClient         litellm.LitellmVirtualKey
	litellmResourceNaming *util.LitellmResourceNaming
	OverrideLiteLLMURL    string
}

// NewVirtualKeyReconciler creates a new VirtualKeyReconciler instance
func NewVirtualKeyReconciler(client client.Client, scheme *runtime.Scheme) *VirtualKeyReconciler {
	return &VirtualKeyReconciler{
		BaseController: &base.BaseController[*authv1alpha1.VirtualKey]{
			Client:         client,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
			ControllerName: "virtualkey",
		},
		LitellmClient:         nil,
		litellmResourceNaming: nil,
		OverrideLiteLLMURL:    "",
	}
}

type ExternalData struct {
	Key      string `json:"key"`
	KeyAlias string `json:"keyAlias"`
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements the single-loop ensure* pattern with finalizer, conditions, and drift sync
func (r *VirtualKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctx, cancel := r.WithTimeout(ctx)
	defer cancel()

	// Instrument the reconcile loop
	r.InstrumentReconcileLoop()
	timer := r.InstrumentReconcileLatency()
	defer timer.ObserveDuration()

	log := log.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	virtualKey := &authv1alpha1.VirtualKey{}
	virtualKey, err := r.FetchResource(ctx, req.NamespacedName, virtualKey)
	if err != nil {
		log.Error(err, "Failed to get VirtualKey")
		r.InstrumentReconcileError()
		return ctrl.Result{}, err
	}
	if virtualKey == nil {
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling external virtual key resource", "virtualKey", virtualKey.Name) // Add timeout to avoid long-running reconciliation
	// Phase 2: Set up connections and clients
	if err := r.ensureConnectionSetup(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to setup connections")
		return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonConnectionError)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if !virtualKey.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, virtualKey)
	}

	// Phase 4: Upsert branch - ensure finalizer
	if err := r.AddFinalizer(ctx, virtualKey, util.FinalizerName); err != nil {
		r.InstrumentReconcileError()
		return ctrl.Result{}, err
	}

	var externalData ExternalData
	// Phase 5: Ensure external resource (create/patch/repair drift)
	if res, err := r.ensureExternal(ctx, virtualKey, &externalData); res.RequeueAfter > 0 || err != nil {
		r.InstrumentReconcileError()
		return res, err
	}

	// Phase 6: Ensure in-cluster children (owned -> GC on delete)
	if err := r.ensureChildren(ctx, virtualKey, &externalData); err != nil {
		return r.HandleCommonErrors(ctx, virtualKey, err)
	}

	// Phase 7: Mark Ready and persist ObservedGeneration
	r.SetSuccessConditions(virtualKey, "VirtualKey is in desired state")
	virtualKey.Status.ObservedGeneration = virtualKey.GetGeneration()
	if err := r.PatchStatus(ctx, virtualKey); err != nil {
		r.InstrumentReconcileError()
		return ctrl.Result{}, err
	}

	// Phase 8: Periodic drift sync (external might change out of band)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *VirtualKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.VirtualKey{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-virtualkey").
		Complete(r)
}

// ensureConnectionSetup configures the LiteLLM client and resource naming
func (r *VirtualKeyReconciler) ensureConnectionSetup(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) error {
	if r.LitellmClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, virtualKey.Spec.ConnectionRef, virtualKey.Namespace)
		if err != nil {
			return err
		}
		r.LitellmClient = litellmConnectionHandler.GetLitellmClient()
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&virtualKey.Spec.ConnectionRef)
	}

	return nil
}

// reconcileDelete handles the deletion branch with idempotent external cleanup
func (r *VirtualKeyReconciler) reconcileDelete(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !r.HasFinalizer(virtualKey, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Set deleting condition and update status
	r.SetCondition(virtualKey, base.CondReady, metav1.ConditionFalse, "Deleting", "VirtualKey is being deleted")
	if err := r.PatchStatus(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to update status during deletion")
		// Continue with deletion even if status update fails
	}

	// Idempotent external cleanup
	if virtualKey.Status.KeyAlias != "" {
		if err := r.LitellmClient.DeleteVirtualKey(ctx, virtualKey.Status.KeyAlias); err != nil {
			log.Error(err, "Failed to delete virtual key from LiteLLM")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonDeleteFailed)
		}
		log.Info("Successfully deleted virtual key from LiteLLM", "keyAlias", virtualKey.Status.KeyAlias)
	}

	// Remove finalizer
	if err := r.RemoveFinalizer(ctx, virtualKey, util.FinalizerName); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonDeleteFailed)
	}

	log.Info("Successfully deleted virtual key", "virtualKey", virtualKey.Name)
	return ctrl.Result{}, nil
}

// ensureExternal manages the external virtual key resource (create/patch/repair drift)
func (r *VirtualKeyReconciler) ensureExternal(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, externalData *ExternalData) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Ensuring external virtual key resource", "virtualKey", virtualKey.Name)

	// Set progressing condition
	r.SetProgressingConditions(virtualKey, "Reconciling virtual key in LiteLLM")
	if err := r.PatchStatus(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to update progressing status")
		// Continue despite status update failure
	}

	desiredVirtualKey, err := r.convertToVirtualKeyRequest(virtualKey)
	if err != nil {
		log.Error(err, "Failed to create virtual key request")
		return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonInvalidSpec)
	}

	observedVirtualKeys, err := r.LitellmClient.GetVirtualKeyFromAlias(ctx, virtualKey.Spec.KeyAlias)
	if err != nil {
		log.Error(err, "Failed to get virtual key from LiteLLM")
		return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonLitellmError)
	}

	var observedVirtualKeyDetails litellm.VirtualKeyResponse

	if len(observedVirtualKeys) == 0 {
		// Create if no external key exists
		log.Info("Creating new virtual key in LiteLLM", "keyAlias", virtualKey.Spec.KeyAlias)
		createResponse, err := r.LitellmClient.GenerateVirtualKey(ctx, &desiredVirtualKey)
		if err != nil {
			log.Error(err, "Failed to create virtual key in LiteLLM")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonLitellmError)
		}

		externalData.Key = createResponse.Key
		externalData.KeyAlias = createResponse.KeyAlias

		secretName := r.litellmResourceNaming.GenerateSecretName(createResponse.KeyAlias)
		r.updateVirtualKeyStatus(virtualKey, createResponse, secretName)
		if err := r.PatchStatus(ctx, virtualKey); err != nil {
			log.Error(err, "Failed to update status after creation")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonReconcileError)
		}
		log.Info("Successfully created virtual key in LiteLLM", "keyAlias", createResponse.KeyAlias)
		return ctrl.Result{}, nil
	} else {
		// Get virtual key details for existing key
		var err error
		observedVirtualKeyDetails, err = r.LitellmClient.GetVirtualKeyInfo(ctx, observedVirtualKeys[0])
		if err != nil {
			log.Error(err, "Failed to get virtual key info from LiteLLM")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonLitellmError)
		}
	}

	updateNeeded := r.LitellmClient.IsVirtualKeyUpdateNeeded(ctx, &observedVirtualKeyDetails, &desiredVirtualKey)
	if updateNeeded {
		log.Info("Repairing drift in LiteLLM", "keyAlias", virtualKey.Spec.KeyAlias)
		// When updating a key, we need to pass the key in the request
		desiredVirtualKey.Key = observedVirtualKeyDetails.Key
		updateResponse, err := r.LitellmClient.UpdateVirtualKey(ctx, &desiredVirtualKey)
		if err != nil {
			log.Error(err, "Failed to update virtual key in LiteLLM")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonLitellmError)
		}

		externalData.Key = updateResponse.Key
		externalData.KeyAlias = updateResponse.KeyAlias
		r.updateVirtualKeyStatus(virtualKey, updateResponse, virtualKey.Status.KeySecretRef)
		if err := r.PatchStatus(ctx, virtualKey); err != nil {
			log.Error(err, "Failed to update status after update")
			return r.HandleErrorRetryable(ctx, virtualKey, err, base.ReasonReconcileError)
		}
		log.Info("Successfully repaired drift in LiteLLM", "keyAlias", virtualKey.Spec.KeyAlias)
	} else {
		log.V(1).Info("Virtual key is up to date in LiteLLM", "keyAlias", virtualKey.Spec.KeyAlias)
		// Still need to populate external data for secret management
		externalData.Key = observedVirtualKeyDetails.Key
		externalData.KeyAlias = virtualKey.Status.KeyAlias
	}

	return ctrl.Result{}, nil
}

// ensureChildren manages in-cluster child resources using CreateOrUpdate pattern
func (r *VirtualKeyReconciler) ensureChildren(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, externalData *ExternalData) error {
	// the VirtualKey is never shown again after the VirtualKey is created, so prevent the secret from being reset to an empty string
	if virtualKey.Status.KeySecretRef == "" || externalData.Key == "" {
		return nil // No secret to create
	}

	secretName := r.litellmResourceNaming.GenerateSecretName(virtualKey.Spec.KeyAlias)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: virtualKey.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		// Set controller reference for garbage collection
		if err := controllerutil.SetControllerReference(virtualKey, secret, r.Scheme); err != nil {
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

// convertToVirtualKeyRequest creates a VirtualKeyRequest from a VirtualKey (isolated for testing)
func (r *VirtualKeyReconciler) convertToVirtualKeyRequest(virtualKey *authv1alpha1.VirtualKey) (litellm.VirtualKeyRequest, error) {
	virtualKeyRequest := litellm.VirtualKeyRequest{
		Aliases:              virtualKey.Spec.Aliases,
		AllowedCacheControls: virtualKey.Spec.AllowedCacheControls,
		AllowedRoutes:        virtualKey.Spec.AllowedRoutes,
		Blocked:              virtualKey.Spec.Blocked,
		BudgetDuration:       virtualKey.Spec.BudgetDuration,
		BudgetID:             virtualKey.Spec.BudgetID,
		Config:               virtualKey.Spec.Config,
		Duration:             virtualKey.Spec.Duration,
		EnforcedParams:       virtualKey.Spec.EnforcedParams,
		Guardrails:           virtualKey.Spec.Guardrails,
		Key:                  virtualKey.Spec.Key,
		KeyAlias:             virtualKey.Spec.KeyAlias,
		MaxParallelRequests:  virtualKey.Spec.MaxParallelRequests,
		Metadata:             util.EnsureMetadata(virtualKey.Spec.Metadata),
		ModelMaxBudget:       virtualKey.Spec.ModelMaxBudget,
		ModelRPMLimit:        virtualKey.Spec.ModelRPMLimit,
		ModelTPMLimit:        virtualKey.Spec.ModelTPMLimit,
		Models:               virtualKey.Spec.Models,
		Permissions:          virtualKey.Spec.Permissions,
		RPMLimit:             virtualKey.Spec.RPMLimit,
		SendInviteEmail:      virtualKey.Spec.SendInviteEmail,
		Tags:                 virtualKey.Spec.Tags,
		TeamID:               virtualKey.Spec.TeamID,
		TPMLimit:             virtualKey.Spec.TPMLimit,
		UserID:               virtualKey.Spec.UserID,
	}

	if virtualKey.Spec.MaxBudget != "" {
		maxBudget, err := strconv.ParseFloat(virtualKey.Spec.MaxBudget, 64)
		if err != nil {
			return litellm.VirtualKeyRequest{}, errors.New("maxBudget: " + err.Error())
		}
		virtualKeyRequest.MaxBudget = maxBudget
	}
	if virtualKey.Spec.SoftBudget != "" {
		softBudget, err := strconv.ParseFloat(virtualKey.Spec.SoftBudget, 64)
		if err != nil {
			return litellm.VirtualKeyRequest{}, errors.New("softBudget: " + err.Error())
		}
		virtualKeyRequest.SoftBudget = softBudget
	}

	return virtualKeyRequest, nil
}

// updateVirtualKeyStatus updates the status of the k8s VirtualKey from the litellm response
func (r *VirtualKeyReconciler) updateVirtualKeyStatus(virtualKey *authv1alpha1.VirtualKey, virtualKeyResponse litellm.VirtualKeyResponse, secretKeyName string) {
	virtualKey.Status.Aliases = virtualKeyResponse.Aliases
	virtualKey.Status.AllowedCacheControls = virtualKeyResponse.AllowedCacheControls
	virtualKey.Status.AllowedRoutes = virtualKeyResponse.AllowedRoutes
	virtualKey.Status.Blocked = virtualKeyResponse.Blocked
	virtualKey.Status.BudgetDuration = virtualKeyResponse.BudgetDuration
	virtualKey.Status.BudgetID = virtualKeyResponse.BudgetID
	virtualKey.Status.BudgetResetAt = virtualKeyResponse.BudgetResetAt
	virtualKey.Status.Config = virtualKeyResponse.Config
	virtualKey.Status.CreatedAt = virtualKeyResponse.CreatedAt
	virtualKey.Status.CreatedBy = virtualKeyResponse.CreatedBy
	virtualKey.Status.Duration = virtualKeyResponse.Duration
	virtualKey.Status.EnforcedParams = virtualKeyResponse.EnforcedParams
	virtualKey.Status.Expires = virtualKeyResponse.Expires
	virtualKey.Status.Guardrails = virtualKeyResponse.Guardrails
	virtualKey.Status.KeyAlias = virtualKeyResponse.KeyAlias
	virtualKey.Status.KeyID = virtualKeyResponse.TokenID
	virtualKey.Status.KeyName = virtualKeyResponse.KeyName
	virtualKey.Status.KeySecretRef = secretKeyName
	virtualKey.Status.LiteLLMBudgetTable = virtualKeyResponse.LiteLLMBudgetTable
	virtualKey.Status.MaxBudget = fmt.Sprintf("%.2f", virtualKeyResponse.MaxBudget)
	virtualKey.Status.MaxParallelRequests = virtualKeyResponse.MaxParallelRequests
	virtualKey.Status.Models = virtualKeyResponse.Models
	virtualKey.Status.Permissions = virtualKeyResponse.Permissions
	virtualKey.Status.RPMLimit = virtualKeyResponse.RPMLimit
	virtualKey.Status.Spend = fmt.Sprintf("%.2f", virtualKeyResponse.Spend)
	virtualKey.Status.Tags = virtualKeyResponse.Tags
	virtualKey.Status.TeamID = virtualKeyResponse.TeamID
	virtualKey.Status.Token = virtualKeyResponse.Token
	virtualKey.Status.TPMLimit = virtualKeyResponse.TPMLimit
	virtualKey.Status.UpdatedAt = virtualKeyResponse.UpdatedAt
	virtualKey.Status.UpdatedBy = virtualKeyResponse.UpdatedBy
	virtualKey.Status.UserID = virtualKeyResponse.UserID
}
