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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	// TODO(user): your logic here
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

	if user.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(user, finalizerName) {
			log.Info("Deleting User: " + user.Status.UserAlias + " from litellm")
			return r.deleteUser(ctx, user)
		}
		return ctrl.Result{}, nil
	}

	if user.Status.Conditions == nil {
		log.Info("Creating User: " + user.Spec.UserAlias + " in litellm")
		return r.createUser(ctx, user)
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

	url := litellmBaseURL + "/user/delete"

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(`{"user_ids": ["`+user.Status.UserID+`"]}`)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+litellmMasterKey)

	defer httpReq.Body.Close()

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Error(err, "Failed to send request to Litellm")
		return ctrl.Result{}, err
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body")
		return ctrl.Result{}, err
	}

	if httpResp.StatusCode != 200 {
		_, err := processLitellmError(log, "Failed to delete User", body)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	controllerutil.RemoveFinalizer(user, finalizerName)
	if err := r.Update(ctx, user); err != nil {
		log.Error(err, "Failed to remove finalizer from User")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createUser creates a new user for the litellm service
func (r *UserReconciler) createUser(ctx context.Context, user *authv1alpha1.User) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	url := litellmBaseURL + "/user/new"

	jsonData, err := buildUserRequestPayload(user.Spec)
	if err != nil {
		log.Error(err, "Failed to build request payload")
		return ctrl.Result{}, err
	}

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+litellmMasterKey)

	defer httpReq.Body.Close()

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Error(err, "Failed to send request to Litellm")
		return ctrl.Result{}, err
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body")
		return ctrl.Result{}, err
	}

	if httpResp.StatusCode != 200 {
		errorJSON, err := processLitellmError(log, "Failed to create User", body)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return ctrl.Result{}, err
		}

		user.Status.Conditions = append(user.Status.Conditions, metav1.Condition{
			Type:               "UserCreated",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "UserCreated",
			Message:            errorJSON.Message,
		})
		if err := r.Status().Update(ctx, user); err != nil {
			log.Error(err, "unable to update User status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Parse the response to get key information
	var response struct {
		CreatedAt           string            `json:"created_at"`
		UpdatedAt           string            `json:"updated_at"`
		UserID              string            `json:"user_id"`
		UserAlias           string            `json:"user_alias"`
		UserEmail           string            `json:"user_email"`
		UserKey             string            `json:"key"`
		UserRole            string            `json:"user_role"`
		Teams               []string          `json:"teams"`
		KeyAlias            string            `json:"key_alias"`
		MaxBudget           float64           `json:"max_budget"`
		ModelMaxBudget      map[string]string `json:"model_max_budget"`
		ModelRPMLimit       map[string]string `json:"model_rpm_limit"`
		ModelTPMLimit       map[string]string `json:"model_tpm_limit"`
		BudgetDuration      string            `json:"budget_duration"`
		Expires             string            `json:"expires"`
		MaxParallelRequests int               `json:"max_parallel_requests"`
		Metadata            map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to parse response body")
		return ctrl.Result{}, err
	}

	secretKeyName := "user-" + response.UserAlias + "-key"

	// Update the status with the key information
	user.Status.CreatedAt = response.CreatedAt
	user.Status.UpdatedAt = response.UpdatedAt
	user.Status.UserID = response.UserID
	user.Status.UserAlias = response.UserAlias
	user.Status.UserEmail = response.UserEmail
	user.Status.UserRole = response.UserRole
	user.Status.Teams = response.Teams
	user.Status.KeyAlias = response.KeyAlias
	user.Status.MaxBudget = fmt.Sprintf("%.2f", response.MaxBudget)
	user.Status.ModelMaxBudget = response.ModelMaxBudget
	user.Status.ModelRPMLimit = response.ModelRPMLimit
	user.Status.ModelTPMLimit = response.ModelTPMLimit
	user.Status.BudgetDuration = response.BudgetDuration
	user.Status.Expires = response.Expires
	user.Status.SecretRef = secretKeyName
	user.Status.MaxParallelRequests = response.MaxParallelRequests
	user.Status.Metadata = response.Metadata

	if err := r.createSecret(ctx, user, secretKeyName, response.UserKey); err != nil {
		log.Error(err, "Failed to create secret")
		return ctrl.Result{}, err
	}

	user.Status.Conditions = append(user.Status.Conditions, metav1.Condition{
		Type:               "UserCreated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "UserCreated",
		Message:            "User created in Litellm",
	})

	if err := r.Status().Update(ctx, user); err != nil {
		log.Error(err, "unable to update User status")
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(user, finalizerName)
	if err := r.Update(ctx, user); err != nil {
		log.Error(err, "Failed to add finalizer to User")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildKeyRequestPayload creates a JSON payload for key generation based on the User spec
func buildUserRequestPayload(spec authv1alpha1.UserSpec) ([]byte, error) {
	payload := make(map[string]interface{})

	// Optional fields - only add if they have values
	if spec.UserID != "" {
		payload["user_id"] = spec.UserID
	}
	if spec.UserAlias != "" {
		payload["user_alias"] = spec.UserAlias
	}
	if spec.UserEmail != "" {
		payload["user_email"] = spec.UserEmail
	}
	if spec.UserRole != "" {
		payload["user_role"] = spec.UserRole
	}
	if spec.SendInviteEmail {
		payload["send_invite_email"] = spec.SendInviteEmail
	}
	if len(spec.Teams) > 0 {
		payload["teams"] = spec.Teams
	}
	if spec.AutoCreateKey {
		payload["auto_create_key"] = spec.AutoCreateKey
	}
	if spec.KeyAlias != "" {
		payload["key_alias"] = spec.KeyAlias
	}
	if spec.SoftBudget != "" {
		payload["soft_budget"] = spec.SoftBudget
	}
	if spec.MaxBudget != "" {
		maxBudget, err := strconv.ParseFloat(spec.MaxBudget, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max_budget value: %v", err)
		}
		payload["max_budget"] = maxBudget
	}
	if spec.ModelMaxBudget != nil {
		payload["model_max_budget"] = spec.ModelMaxBudget
	}
	if spec.ModelRPMLimit != nil {
		payload["model_rpm_limit"] = spec.ModelRPMLimit
	}
	if spec.ModelTPMLimit != nil {
		payload["model_tpm_limit"] = spec.ModelTPMLimit
	}
	payload["budget_duration"] = spec.BudgetDuration
	if spec.BudgetDuration != "" {
		payload["budget_duration"] = spec.BudgetDuration
	}
	if len(spec.Models) > 0 {
		payload["models"] = spec.Models
	}
	if spec.MaxParallelRequests != 0 {
		payload["max_parallel_requests"] = spec.MaxParallelRequests
	}
	if spec.Metadata != nil {
		payload["metadata"] = spec.Metadata
	}

	return json.Marshal(payload)
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
			"Key": []byte(key),
		},
	}

	return r.Create(ctx, secret)
}
