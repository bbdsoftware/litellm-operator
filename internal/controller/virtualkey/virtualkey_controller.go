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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// VirtualKeyReconciler reconciles a VirtualKey object
type VirtualKeyReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	LitellmClient         litellm.LitellmVirtualKey
	litellmResourceNaming *util.LitellmResourceNaming
	OverrideLiteLLMURL    string
}

// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.litellm.ai,resources=virtualkeys/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualKey object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *VirtualKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	virtualKey := &authv1alpha1.VirtualKey{}
	if err := r.Get(ctx, req.NamespacedName, virtualKey); err != nil {
		// If the custom resource is not found then, it usually means that it was deleted or not created
		// In this way, we will stop the reconciliation
		if apierrors.IsNotFound(err) {
			log.Info("VirtualKey resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get VirtualKey")
		return ctrl.Result{}, err
	}

	// Initialize connection handler if not already done
	litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, virtualKey.Spec.ConnectionRef, virtualKey.Namespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	r.LitellmClient = litellmConnectionHandler.GetLitellmClient()

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(&virtualKey.Spec.ConnectionRef)
	}

	// If the VirtualKey is being deleted, delete the key from litellm
	if virtualKey.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(virtualKey, util.FinalizerName) {
			log.Info("Deleting VirtualKey: " + virtualKey.Status.KeyAlias + " from litellm")
			return r.deleteVirtualKey(ctx, virtualKey)
		}
		return ctrl.Result{}, nil
	}

	exists, err := r.LitellmClient.CheckVirtualKeyExists(ctx, virtualKey.Spec.KeyAlias)
	if err != nil {
		log.Error(err, "Failed to check if VirtualKey exists")
		if _, updateErr := r.updateConditions(ctx, virtualKey, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "UnableToCheckVirtualKeyExists",
			Message: err.Error(),
		}); updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	if !exists {
		// Key does not exist, generate it
		log.Info("Generating new VirtualKey: " + virtualKey.Spec.KeyAlias + " in litellm")
		return r.generateVirtualKey(ctx, virtualKey)
	}

	// sync key if required
	err = r.syncVirtualKey(ctx, virtualKey)
	if err != nil {
		log.Error(err, "Failed to sync virtual key")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.VirtualKey{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// updateConditions updates the VirtualKey status with the given condition
func (r *VirtualKeyReconciler) updateConditions(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if meta.SetStatusCondition(&virtualKey.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, virtualKey); err != nil {
			log.Error(err, "unable to update VirtualKey status with condition")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteVirtualKey handles the deletion of a virtual key from the litellm service
func (r *VirtualKeyReconciler) deleteVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.LitellmClient.DeleteVirtualKey(ctx, virtualKey.Status.KeyAlias); err != nil {
		return r.updateConditions(ctx, virtualKey, metav1.Condition{
			Type:    "Deleted",
			Status:  metav1.ConditionFalse,
			Reason:  "DeletionFailed",
			Message: err.Error(),
		})
	}

	controllerutil.RemoveFinalizer(virtualKey, util.FinalizerName)
	if err := r.Update(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Deleted VirtualKey: " + virtualKey.Status.KeyAlias + " from litellm")
	return ctrl.Result{}, nil
}

// generateVirtualKey generates a new virtual key for the litellm service
func (r *VirtualKeyReconciler) generateVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	virtualKeyRequest, err := createVirtualKeyRequest(virtualKey)
	if err != nil {
		if _, err := r.updateConditions(ctx, virtualKey, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	virtualKeyResponse, err := r.LitellmClient.GenerateVirtualKey(ctx, &virtualKeyRequest)
	if err != nil {
		return r.updateConditions(ctx, virtualKey, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "LitellmError",
			Message: err.Error(),
		})
	}

	resourceNaming := util.NewLitellmResourceNaming(&virtualKey.Spec.ConnectionRef)
	secretName := resourceNaming.GenerateSecretName(virtualKeyResponse.KeyAlias)

	updateVirtualKeyStatus(virtualKey, virtualKeyResponse, secretName)
	_, err = r.updateConditions(ctx, virtualKey, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "LitellmSuccess",
		Message: "VirtualKey generated in Litellm",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(virtualKey, util.FinalizerName)
	if err := r.Update(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	if err := r.createSecret(ctx, virtualKey, secretName, virtualKeyResponse.Key); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
	}

	log.Info("Generated VirtualKey: " + virtualKey.Status.KeyAlias + " in litellm")
	return ctrl.Result{}, nil
}

// createSecret stores the secret key in a Kubernetes Secret that is owned by the VirtualKey
func (r *VirtualKeyReconciler) createSecret(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, secretName string, key string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: virtualKey.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "auth.litellm.ai/v1alpha1",
					Kind:       "VirtualKey",
					Name:       virtualKey.Name,
					UID:        virtualKey.UID,
				},
			},
		},
		Data: map[string][]byte{
			"key": []byte(key),
		},
	}

	return r.Create(ctx, secret)
}

// syncVirtualKey syncs the virtual key with the litellm service should changes be detected
func (r *VirtualKeyReconciler) syncVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) error {
	log := log.FromContext(ctx)

	key, err := r.getKeyFromSecret(ctx, virtualKey)
	if err != nil {
		log.Error(err, "Failed to get key from secret")
		return err
	}

	virtualKeyRequest, err := createVirtualKeyRequest(virtualKey)
	if err != nil {
		log.Error(err, "Failed to create virtual key request")
		return err
	}

	virtualKeyResponse, err := r.LitellmClient.GetVirtualKey(ctx, key)
	if err != nil {
		log.Error(err, "Failed to get virtual key from litellm")
		return err
	}

	if r.LitellmClient.IsVirtualKeyUpdateNeeded(ctx, &virtualKeyResponse, &virtualKeyRequest) {
		log.Info("Updating VirtualKey: " + virtualKey.Spec.KeyAlias + " in litellm")
		// When updating a key, we need to pass the key in the request but this usually resides in the Secret
		virtualKeyRequest.Key = key

		updatedResponse, err := r.LitellmClient.UpdateVirtualKey(ctx, &virtualKeyRequest)
		if err != nil {
			log.Error(err, "Failed to update virtual key in litellm")
			return err
		}

		updateVirtualKeyStatus(virtualKey, updatedResponse, virtualKey.Status.KeySecretRef)

		if err := r.Status().Update(ctx, virtualKey); err != nil {
			log.Error(err, "Failed to update VirtualKey status")
			return err
		} else {
			log.Info("Updated VirtualKey: " + virtualKey.Spec.KeyAlias + " in litellm")
		}
	}

	return nil
}

// getKeyFromSecret gets the key from the secret associated with the VirtualKey
func (r *VirtualKeyReconciler) getKeyFromSecret(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (string, error) {

	namespacedName := types.NamespacedName{
		Name:      r.litellmResourceNaming.GenerateSecretName(virtualKey.Spec.KeyAlias),
		Namespace: virtualKey.Namespace,
	}
	var secret corev1.Secret
	if err := r.Get(ctx, namespacedName, &secret); err != nil {
		return "", err
	}
	return string(secret.Data["key"]), nil
}

// createVirtualKeyRequest creates a VirtualKeyRequest from a VirtualKey
func createVirtualKeyRequest(virtualKey *authv1alpha1.VirtualKey) (litellm.VirtualKeyRequest, error) {
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
func updateVirtualKeyStatus(virtualKey *authv1alpha1.VirtualKey, virtualKeyResponse litellm.VirtualKeyResponse, secretKeyName string) {
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
