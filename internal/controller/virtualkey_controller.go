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

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualKeyReconciler reconciles a VirtualKey object
type VirtualKeyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	litellm.LitellmVirtualKey
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

	if virtualKey.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(virtualKey, finalizerName) {
			log.Info("Deleting VirtualKey: " + virtualKey.Status.KeyAlias + " from litellm")
			return r.deleteVirtualKey(ctx, virtualKey)
		}
		return ctrl.Result{}, nil
	}

	if virtualKey.Status.Conditions == nil {
		log.Info("Generating new VirtualKey: " + virtualKey.Spec.KeyAlias + " in litellm")
		return r.generateVirtualKey(ctx, virtualKey)
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

// appendCondition appends a condition to the VirtualKey status and updates the VirtualKey
func (r *VirtualKeyReconciler) appendCondition(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, condition metav1.Condition) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	virtualKey.Status.Conditions = append(virtualKey.Status.Conditions, condition)
	if err := r.Status().Update(ctx, virtualKey); err != nil {
		log.Error(err, "unable to update VirtualKey status with condition")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deleteVirtualKey handles the deletion of a virtual key from the litellm service
func (r *VirtualKeyReconciler) deleteVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if err := r.DeleteVirtualKey(ctx, virtualKey.Status.KeyAlias); err != nil {
		return r.appendCondition(ctx, virtualKey, metav1.Condition{
			Type:               "DeleteVirtualKey",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeleteVirtualKeyFailure",
			Message:            err.Error(),
		})
	}

	controllerutil.RemoveFinalizer(virtualKey, finalizerName)
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

	generateVirtualKeyResponse, err := r.GenerateVirtualKey(ctx, &litellm.VirtualKeyRequest{
		KeyAlias:       virtualKey.Spec.KeyAlias,
		UserID:         virtualKey.Spec.UserID,
		TeamID:         virtualKey.Spec.TeamID,
		MaxBudget:      virtualKey.Spec.MaxBudget,
		BudgetDuration: virtualKey.Spec.BudgetDuration,
		Models:         virtualKey.Spec.Models,
		Metadata:       ensureMetadata(virtualKey.Spec.Metadata),
	})

	if err != nil {
		return r.appendCondition(ctx, virtualKey, metav1.Condition{
			Type:               "GenerateVirtualKey",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "GenerateVirtualKeyFailure",
			Message:            err.Error(),
		})
	}

	secretKeyName := "virtual-key-" + generateVirtualKeyResponse.KeyAlias

	virtualKey.Status.CreatedAt = generateVirtualKeyResponse.CreatedAt
	virtualKey.Status.UpdatedAt = generateVirtualKeyResponse.UpdatedAt
	virtualKey.Status.Expires = generateVirtualKeyResponse.Expires
	virtualKey.Status.KeyAlias = generateVirtualKeyResponse.KeyAlias
	virtualKey.Status.KeyID = generateVirtualKeyResponse.TokenID
	virtualKey.Status.KeyName = generateVirtualKeyResponse.KeyName
	virtualKey.Status.MaxBudget = fmt.Sprintf("%.2f", generateVirtualKeyResponse.MaxBudget)
	virtualKey.Status.BudgetDuration = generateVirtualKeyResponse.BudgetDuration
	virtualKey.Status.Models = generateVirtualKeyResponse.Models
	virtualKey.Status.UserID = generateVirtualKeyResponse.UserID
	virtualKey.Status.SecretRef = secretKeyName
	virtualKey.Status.Metadata = generateVirtualKeyResponse.Metadata

	controllerutil.AddFinalizer(virtualKey, finalizerName)

	if _, err := r.appendCondition(ctx, virtualKey, metav1.Condition{
		Type:               "GenerateVirtualKey",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "GenerateVirtualKeySuccess",
		Message:            "VirtualKey generated in Litellm",
	}); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createSecret(ctx, virtualKey, secretKeyName, generateVirtualKeyResponse.SecretKey); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
	}

	log.Info("Created VirtualKey: " + virtualKey.Status.KeyAlias + " in litellm")
	return ctrl.Result{}, nil
}

// createSecret stores the secret key in a Kubernetes Secret that is owned by the VirtualKey
func (r *VirtualKeyReconciler) createSecret(ctx context.Context, virtualKey *authv1alpha1.VirtualKey, secretName string, secretKey string) error {
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
			"SecretKey": []byte(secretKey),
		},
	}

	return r.Create(ctx, secret)
}
