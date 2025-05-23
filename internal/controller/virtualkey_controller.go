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
	"encoding/json"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualKeyReconciler reconciles a VirtualKey object
type VirtualKeyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

// deleteVirtualKey handles the deletion of a virtual key from the litellm service
func (r *VirtualKeyReconciler) deleteVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	url := litellmBaseURL + "/key/delete"

	_, err := makeLitellmRequest(ctx, "POST", url, []byte(`{"key_aliases": ["`+virtualKey.Status.KeyAlias+`"]}`))
	if err != nil {
		log.Error(err, "Failed to delete key")
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(virtualKey, finalizerName)
	if err := r.Update(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to remove finalizer from VirtualKey")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// generateVirtualKey generates a new virtual key for the litellm service
func (r *VirtualKeyReconciler) generateVirtualKey(ctx context.Context, virtualKey *authv1alpha1.VirtualKey) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	url := litellmBaseURL + "/key/generate"

	jsonData, err := buildKeyRequestPayload(virtualKey.Spec)
	if err != nil {
		log.Error(err, "Failed to build request payload")
		return ctrl.Result{}, err
	}

	body, err := makeLitellmRequest(ctx, "POST", url, jsonData)
	if err != nil {
		log.Error(err, "Failed to generate key")

		virtualKey.Status.Conditions = append(virtualKey.Status.Conditions, metav1.Condition{
			Type:               "KeyGenerated",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "KeyGenerated",
			Message:            err.Error(),
		})
		if err := r.Status().Update(ctx, virtualKey); err != nil {
			log.Error(err, "unable to update VirtualKey status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Parse the response to get key information
	var response struct {
		CreatedAt      string            `json:"created_at"`
		UpdatedAt      string            `json:"updated_at"`
		BudgetDuration string            `json:"budget_duration"`
		Expires        string            `json:"expires"`
		KeyAlias       string            `json:"key_alias"`
		KeyName        string            `json:"key_name"`
		MaxBudget      float64           `json:"max_budget"`
		Models         []string          `json:"models"`
		SecretKey      string            `json:"key"`
		TokenID        string            `json:"token_id"`
		UserID         string            `json:"user_id"`
		Metadata       map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to parse response body")
		return ctrl.Result{}, err
	}

	secretKeyName := "virtual-key-" + response.KeyAlias

	// Update the status with the key information
	virtualKey.Status.CreatedAt = response.CreatedAt
	virtualKey.Status.UpdatedAt = response.UpdatedAt
	virtualKey.Status.BudgetDuration = response.BudgetDuration
	virtualKey.Status.Expires = response.Expires
	virtualKey.Status.KeyAlias = response.KeyAlias
	virtualKey.Status.KeyID = response.TokenID
	virtualKey.Status.KeyName = response.KeyName
	virtualKey.Status.MaxBudget = fmt.Sprintf("%.2f", response.MaxBudget)
	virtualKey.Status.Models = response.Models
	virtualKey.Status.UserID = response.UserID
	virtualKey.Status.SecretRef = secretKeyName
	virtualKey.Status.Metadata = response.Metadata

	if err := r.createSecret(ctx, virtualKey, secretKeyName, response.SecretKey); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
	}

	virtualKey.Status.Conditions = append(virtualKey.Status.Conditions, metav1.Condition{
		Type:               "KeyGenerated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "KeyGenerated",
		Message:            "Key generated in Litellm",
	})

	if err := r.Status().Update(ctx, virtualKey); err != nil {
		log.Error(err, "unable to update VirtualKey status")
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(virtualKey, finalizerName)
	if err := r.Update(ctx, virtualKey); err != nil {
		log.Error(err, "Failed to add finalizer to VirtualKey")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildKeyRequestPayload creates a JSON payload for key generation based on the VirtualKey spec
func buildKeyRequestPayload(spec authv1alpha1.VirtualKeySpec) ([]byte, error) {
	payload := make(map[string]interface{})

	// Required fields
	payload["key_alias"] = spec.KeyAlias
	payload["user_id"] = spec.UserID

	// Optional fields - only add if they have values
	if spec.TeamID != "" {
		payload["team_id"] = spec.TeamID
	}
	if spec.MaxBudget != "" {
		maxBudget, err := strconv.ParseFloat(spec.MaxBudget, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_budget value: %v", err)
		}
		payload["max_budget"] = maxBudget
	}
	if spec.BudgetDuration != "" {
		payload["budget_duration"] = spec.BudgetDuration
	}
	if len(spec.Models) > 0 {
		payload["models"] = spec.Models
	}
	operatorMetadata := map[string]string{
		"managed_by": "litellm-operator",
	}
	if spec.Metadata != nil {
		for k, v := range spec.Metadata {
			operatorMetadata[k] = v
		}
	}
	payload["metadata"] = operatorMetadata

	return json.Marshal(payload)
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
