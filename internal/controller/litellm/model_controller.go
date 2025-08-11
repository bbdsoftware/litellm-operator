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

package litellm

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	LitellmClient     *litellm.LitellmClient
	connectionHandler *common.ConnectionHandler
}

// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	model := &litellmv1alpha1.Model{}
	err := r.Get(ctx, req.NamespacedName, model)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("Model resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Model")
		return ctrl.Result{}, err
	}

	// Initialize connection handler if not already done
	if r.connectionHandler == nil {
		r.connectionHandler = common.NewConnectionHandler(r.Client)
	}

	// Get connection details
	connectionDetails, err := r.connectionHandler.GetConnectionDetailsFromLitellmRef(ctx, model.Spec.ConnectionRef, model.Namespace)
	if err != nil {
		// log.Error(err, "Failed to get connection details")
		// if _, updateErr := r.updateConditions(ctx, model, metav1.Condition{
		// 	Type:               "Ready",
		// 	Status:             metav1.ConditionFalse,
		// 	LastTransitionTime: metav1.Now(),
		// 	Reason:             "ConnectionError",
		// 	Message:            err.Error(),
		// }); updateErr != nil {
		// 	log.Error(updateErr, "Failed to update conditions")
		// }
		return ctrl.Result{}, err
	}

	// Configure the LiteLLM client with connection details only if not already set (for testing)
	if r.LitellmClient == nil {
		r.LitellmClient = common.ConfigureLitellmClient(connectionDetails)
	}

	// Check if the resource is being deleted
	if !model.DeletionTimestamp.IsZero() {
		// Resource is being deleted, handle cleanup
		return r.handleDeletion(ctx, model)
	}

	// Convert Kubernetes Model to LiteLLM ModelRequest
	modelRequest, err := r.convertToModelRequest(model)
	if err != nil {
		log.Error(err, "Failed to convert Model to ModelRequest")
		return ctrl.Result{}, err
	}

	// Try to get the existing model from LiteLLM
	existingModel, err := r.LitellmClient.GetModel(ctx, modelRequest.ModelName)
	if err != nil {
		// Model doesn't exist, create it
		log.Info("Model not found in LiteLLM, creating new model")
		return r.handleCreation(ctx, model, modelRequest)
	}

	// Model exists, check if update is needed
	if r.LitellmClient.IsModelUpdateNeeded(ctx, &existingModel, modelRequest) {
		log.Info("Model needs update, updating in LiteLLM")
		return r.handleUpdate(ctx, model, modelRequest)
	}

	log.Info("Model is up to date")
	return ctrl.Result{}, nil
}

// handleCreation handles the creation of a new model
func (r *ModelReconciler) handleCreation(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	_, err := r.LitellmClient.CreateModel(ctx, modelRequest)
	if err != nil {
		log.Error(err, "Failed to create model in LiteLLM")
		return ctrl.Result{}, err
	}

	log.Info("Successfully created model in LiteLLM")
	return ctrl.Result{}, nil
}

// handleUpdate handles the update of an existing model
func (r *ModelReconciler) handleUpdate(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	_, err := r.LitellmClient.UpdateModel(ctx, modelRequest)
	if err != nil {
		log.Error(err, "Failed to update model in LiteLLM")
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated model in LiteLLM")
	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of a model
func (r *ModelReconciler) handleDeletion(ctx context.Context, model *litellmv1alpha1.Model) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Try to delete the model from LiteLLM
	err := r.LitellmClient.DeleteModel(ctx, model.Spec.ModelName)
	if err != nil {
		log.Error(err, "Failed to delete model from LiteLLM")
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted model from LiteLLM")
	return ctrl.Result{}, nil
}

// convertToModelRequest converts a Kubernetes Model to a LiteLLM ModelRequest
func (r *ModelReconciler) convertToModelRequest(model *litellmv1alpha1.Model) (*litellm.ModelRequest, error) {
	modelRequest := &litellm.ModelRequest{
		ModelName: model.Spec.ModelName,
	}

	// Convert LiteLLMParams
	if !reflect.DeepEqual(model.Spec.LiteLLMParams, litellmv1alpha1.LiteLLMParams{}) {
		// Convert int64 to int for TPM, RPM, MaxRetries
		// tpm := int(model.Spec.LiteLLMParams.TPM)
		// rpm := int(model.Spec.LiteLLMParams.RPM)
		// maxRetries := int(model.Spec.LiteLLMParams.MaxRetries)

		// Convert int64 to float64 for MaxFileSizeMB

		litellmParams := &litellm.UpdateLiteLLMParams{
			// InputCostPerToken:              &model.Spec.LiteLLMParams.InputCostPerToken,
			// OutputCostPerToken:             &model.Spec.LiteLLMParams.OutputCostPerToken,
			// InputCostPerSecond:             &model.Spec.LiteLLMParams.InputCostPerSecond,
			// OutputCostPerSecond:            &model.Spec.LiteLLMParams.OutputCostPerSecond,
			// InputCostPerPixel:              &model.Spec.LiteLLMParams.InputCostPerPixel,
			// OutputCostPerPixel:             &model.Spec.LiteLLMParams.OutputCostPerPixel,
			APIKey:  &model.Spec.LiteLLMParams.APIKey,
			APIBase: &model.Spec.LiteLLMParams.APIBase,
			// APIVersion:                     &model.Spec.LiteLLMParams.APIVersion,
			// VertexProject:                  &model.Spec.LiteLLMParams.VertexProject,
			// VertexLocation:                 &model.Spec.LiteLLMParams.VertexLocation,
			// RegionName:                     &model.Spec.LiteLLMParams.RegionName,
			// AWSAccessKeyID:                 &model.Spec.LiteLLMParams.AWSAccessKeyID,
			// AWSSecretAccessKey:             &model.Spec.LiteLLMParams.AWSSecretAccessKey,
			// AWSRegionName:                  &model.Spec.LiteLLMParams.AWSRegionName,
			// WatsonXRegionName:              &model.Spec.LiteLLMParams.WatsonXRegionName,
			CustomLLMProvider: &model.Spec.LiteLLMParams.CustomLLMProvider,
			// TPM:                            &tpm,
			// RPM:                            &rpm,
			// MaxRetries:                     &maxRetries,
			// Organisation:                   &model.Spec.LiteLLMParams.Organisation,
			// LiteLLMCredentialName:          &model.Spec.LiteLLMParams.LiteLLMCredentialName,
			// LiteLLMTraceID:                 &model.Spec.LiteLLMParams.LiteLLMTraceID,
			// MaxFileSizeMB:                  &model.Spec.LiteLLMParams.MaxFileSizeMB,
			// MaxBudget:                      &model.Spec.LiteLLMParams.MaxBudget,
			// BudgetDuration:                 &model.Spec.LiteLLMParams.BudgetDuration,
			// UseInPassThrough:               &model.Spec.LiteLLMParams.UseInPassThrough,
			// UseLiteLLMProxy:                &model.Spec.LiteLLMParams.UseLiteLLMProxy,
			// MergeReasoningContentInChoices: &model.Spec.LiteLLMParams.MergeReasoningContentInChoices,
			// AutoRouterConfigPath:           &model.Spec.LiteLLMParams.AutoRouterConfigPath,
			// AutoRouterConfig:               &model.Spec.LiteLLMParams.AutoRouterConfig,
			// AutoRouterDefaultModel:         &model.Spec.LiteLLMParams.AutoRouterDefaultModel,
			// AutoRouterEmbeddingModel:       &model.Spec.LiteLLMParams.AutoRouterEmbeddingModel,
			Model: &model.Spec.LiteLLMParams.Model,
		}

		// Handle timeout fields
		if model.Spec.LiteLLMParams.Timeout != 0 {
			litellmParams.Timeout = model.Spec.LiteLLMParams.Timeout
		}
		if model.Spec.LiteLLMParams.StreamTimeout != 0 {
			litellmParams.StreamTimeout = model.Spec.LiteLLMParams.StreamTimeout
		}

		// Handle VertexCredentials
		if model.Spec.LiteLLMParams.VertexCredentials != "" {
			litellmParams.VertexCredentials = model.Spec.LiteLLMParams.VertexCredentials
		}

		// Handle ConfigurableClientsideAuthParams
		if len(model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams) > 0 {
			litellmParams.ConfigurableClientsideAuthParams = make([]interface{}, len(model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams))
			for i, param := range model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams {
				litellmParams.ConfigurableClientsideAuthParams[i] = param
			}
		}

		// // Handle ModelInfo
		// if model.Spec.LiteLLMParams.ModelInfo.Raw != nil {
		// 	litellmParams.ModelInfo = model.Spec.LiteLLMParams.ModelInfo
		// } else {
		// 	litellmParams.ModelInfo = litellm.NewModelInfo()
		// }

		// // Handle MockResponse
		// if model.Spec.LiteLLMParams.MockResponse != "" {
		// 	litellmParams.MockResponse = model.Spec.LiteLLMParams.MockResponse
		// }

		// // Handle AdditionalProps
		// if model.Spec.LiteLLMParams.AdditionalProps.Raw != nil {
		// 	litellmParams.AdditionalProperties = make(map[string]interface{})
		// 	// Convert AdditionalProps to map[string]interface{} if needed
		// }

		modelRequest.LiteLLMParams = litellmParams
	}

	// // Convert ModelInfo
	// if !reflect.DeepEqual(model.Spec.ModelInfo, litellmv1alpha1.ModelInfo{}) {
	// 	modelInfo := &litellm.ModelInfo{
	// 		ID:                  &model.Spec.ModelInfo.ID,
	// 		DBModel:             &model.Spec.ModelInfo.DBModel,
	// 		UpdatedBy:           &model.Spec.ModelInfo.UpdatedBy,
	// 		CreatedBy:           &model.Spec.ModelInfo.CreatedBy,
	// 		BaseModel:           &model.Spec.ModelInfo.BaseModel,
	// 		Tier:                &model.Spec.ModelInfo.Tier,
	// 		TeamID:              &model.Spec.ModelInfo.TeamID,
	// 		TeamPublicModelName: &model.Spec.ModelInfo.TeamPublicModelName,
	// 	}

	// 	// Handle timestamp fields
	// 	if !model.Spec.ModelInfo.UpdatedAt.IsZero() {
	// 		updatedAt := model.Spec.ModelInfo.UpdatedAt.Format("2006-01-02T15:04:05Z")
	// 		modelInfo.UpdatedAt = &updatedAt
	// 	}
	// 	if !model.Spec.ModelInfo.CreatedAt.IsZero() {
	// 		createdAt := model.Spec.ModelInfo.CreatedAt.Format("2006-01-02T15:04:05Z")
	// 		modelInfo.CreatedAt = &createdAt
	// 	}

	// 	// Handle AdditionalProps
	// 	if model.Spec.ModelInfo.AdditionalProps.Raw != nil {
	// 		modelInfo.AdditionalProperties = make(map[string]interface{})
	// 		// Convert AdditionalProps to map[string]interface{} if needed
	// 	}

	// 	modelRequest.ModelInfo = modelInfo
	// }

	return modelRequest, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&litellmv1alpha1.Model{}).
		Named("litellm-model").
		Complete(r)
}
