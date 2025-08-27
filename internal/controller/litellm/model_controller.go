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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	LitellmModelClient    litellm.LitellmModel
	connectionHandler     *common.ConnectionHandler
	litellmResourceNaming *util.LitellmResourceNaming
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
		log.Error(err, "Failed to get connection details")
		if _, updateErr := r.updateConditions(ctx, model, metav1.Condition{
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
	if r.LitellmModelClient == nil {
		r.LitellmModelClient = common.ConfigureLitellmClient(connectionDetails)
	}

	// Check if the resource is being deleted
	if model.GetDeletionTimestamp() != nil {
		// Resource is being deleted, handle cleanup
		if controllerutil.ContainsFinalizer(model, util.FinalizerName) {
			log.Info("Model:" + model.Name + " is being deleted from litellm")
			return r.handleDeletion(ctx, model)
		}
		return ctrl.Result{}, nil

	}

	// Convert Kubernetes Model to LiteLLM ModelRequest
	modelRequest, err := r.convertToModelRequest(model)
	if err != nil {
		log.Error(err, "Failed to convert Model to ModelRequest")
		if _, err := r.updateConditions(ctx, model, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Try to get the existing model from LiteLLM service using model ID
	if model.Status.ModelId != nil && *model.Status.ModelId != "" {
		existingModel, err := r.LitellmModelClient.GetModel(ctx, *model.Status.ModelId)
		if err != nil {
			log.Error(err, "Failed to get existing model from LiteLLM")
			if _, updateErr := r.updateConditions(ctx, model, metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "UnableToCheckModelExists",
				Message: err.Error(),
			}); updateErr != nil {
				log.Error(updateErr, "Failed to update conditions")
			}
			return ctrl.Result{}, err
		}
		// Model exists, check if update is needed
		if r.LitellmModelClient.IsModelUpdateNeeded(ctx, &existingModel, modelRequest) {
			log.Info("Model needs update, updating in LiteLLM")
			return r.handleUpdate(ctx, model, modelRequest)
		}
	}

	// Model doesn't exist, create it
	return r.handleCreation(ctx, model, modelRequest)
}

// handleCreation handles the creation of a new model
func (r *ModelReconciler) handleCreation(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	modelResponse, err := r.LitellmModelClient.CreateModel(ctx, modelRequest)
	if err != nil {
		if _, err := r.updateConditions(ctx, model, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "LitellmError",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	updateModelStatus(model, &modelResponse)
	//update status
	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update Model status")
		return ctrl.Result{}, err
	}

	_, err = r.updateConditions(ctx, model, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "LitellmSuccess",
		Message: "Model created in Litellm",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(model, util.FinalizerName)
	if err := r.Update(ctx, model); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully created model " + model.Name + " in LiteLLM")
	return ctrl.Result{}, nil
}

func updateModelStatus(model *litellmv1alpha1.Model, modelResponse *litellm.ModelResponse) {

	model.Status.ModelName = &modelResponse.ModelName
	model.Status.ModelId = modelResponse.ModelInfo.ID
	if modelResponse.LiteLLMParams != nil && modelResponse.LiteLLMParams != (&litellm.UpdateLiteLLMParams{}) {
		model.Status.LiteLLMParams = &litellmv1alpha1.LiteLLMParams{

			ApiBase:                        modelResponse.LiteLLMParams.ApiBase,
			RegionName:                     modelResponse.LiteLLMParams.RegionName,
			CustomLLMProvider:              modelResponse.LiteLLMParams.CustomLLMProvider,
			TPM:                            modelResponse.LiteLLMParams.TPM,
			RPM:                            modelResponse.LiteLLMParams.RPM,
			MaxRetries:                     modelResponse.LiteLLMParams.MaxRetries,
			Organization:                   modelResponse.LiteLLMParams.Organization,
			UseInPassThrough:               modelResponse.LiteLLMParams.UseInPassThrough,
			UseLiteLLMProxy:                modelResponse.LiteLLMParams.UseLiteLLMProxy,
			MergeReasoningContentInChoices: modelResponse.LiteLLMParams.MergeReasoningContentInChoices,
			Model:                          modelResponse.LiteLLMParams.Model,
		}
	}

}

// handleUpdate handles the update of an existing model
func (r *ModelReconciler) handleUpdate(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	_, err := r.LitellmModelClient.UpdateModel(ctx, modelRequest)
	if err != nil {
		log.Error(err, "Failed to update model in LiteLLM", "modelName", model.Spec.ModelName)
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated model in LiteLLM", "modelName", model.Spec.ModelName)
	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of a model
func (r *ModelReconciler) handleDeletion(ctx context.Context, model *litellmv1alpha1.Model) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Try to delete the model from LiteLLM
	err := r.LitellmModelClient.DeleteModel(ctx, *model.Status.ModelId)
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

		litellmParams := &litellm.UpdateLiteLLMParams{
			ApiKey:                         model.Spec.LiteLLMParams.ApiKey,
			ApiBase:                        model.Spec.LiteLLMParams.ApiBase,
			ApiVersion:                     model.Spec.LiteLLMParams.ApiVersion,
			VertexProject:                  model.Spec.LiteLLMParams.VertexProject,
			VertexLocation:                 model.Spec.LiteLLMParams.VertexLocation,
			RegionName:                     model.Spec.LiteLLMParams.RegionName,
			AwsAccessKeyID:                 model.Spec.LiteLLMParams.AwsAccessKeyID,
			AwsSecretAccessKey:             model.Spec.LiteLLMParams.AwsSecretAccessKey,
			AwsRegionName:                  model.Spec.LiteLLMParams.AwsRegionName,
			WatsonXRegionName:              model.Spec.LiteLLMParams.WatsonXRegionName,
			CustomLLMProvider:              model.Spec.LiteLLMParams.CustomLLMProvider,
			TPM:                            model.Spec.LiteLLMParams.TPM,
			RPM:                            model.Spec.LiteLLMParams.RPM,
			MaxRetries:                     model.Spec.LiteLLMParams.MaxRetries,
			Organization:                   model.Spec.LiteLLMParams.Organization,
			LiteLLMCredentialName:          model.Spec.LiteLLMParams.LiteLLMCredentialName,
			LiteLLMTraceID:                 model.Spec.LiteLLMParams.LiteLLMTraceID,
			MaxFileSizeMB:                  model.Spec.LiteLLMParams.MaxFileSizeMB,
			BudgetDuration:                 model.Spec.LiteLLMParams.BudgetDuration,
			UseInPassThrough:               model.Spec.LiteLLMParams.UseInPassThrough,
			UseLiteLLMProxy:                model.Spec.LiteLLMParams.UseLiteLLMProxy,
			MergeReasoningContentInChoices: model.Spec.LiteLLMParams.MergeReasoningContentInChoices,
			AutoRouterConfigPath:           model.Spec.LiteLLMParams.AutoRouterConfigPath,
			AutoRouterConfig:               model.Spec.LiteLLMParams.AutoRouterConfig,
			AutoRouterDefaultModel:         model.Spec.LiteLLMParams.AutoRouterDefaultModel,
			AutoRouterEmbeddingModel:       model.Spec.LiteLLMParams.AutoRouterEmbeddingModel,
			Model:                          model.Spec.LiteLLMParams.Model,
		}

		// Handle timeout fields
		if model.Spec.LiteLLMParams.Timeout != nil && *model.Spec.LiteLLMParams.Timeout != 0 {
			litellmParams.Timeout = model.Spec.LiteLLMParams.Timeout
		}
		if model.Spec.LiteLLMParams.StreamTimeout != nil && *model.Spec.LiteLLMParams.StreamTimeout != 0 {
			litellmParams.StreamTimeout = model.Spec.LiteLLMParams.StreamTimeout
		}

		// Handle VertexCredentials
		if model.Spec.LiteLLMParams.VertexCredentials != nil && *model.Spec.LiteLLMParams.VertexCredentials != "" {
			litellmParams.VertexCredentials = model.Spec.LiteLLMParams.VertexCredentials
		}

		// Handle ConfigurableClientsideAuthParams
		if model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams != nil && len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams) > 0 {
			litellmParams.ConfigurableClientsideAuthParams = make([]interface{}, len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams))
			for i, param := range *model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams {
				litellmParams.ConfigurableClientsideAuthParams[i] = param
			}
		}

		// Handle ModelInfo
		// if model.Spec.ModelInfo != nil && model.Spec.LiteLLMParams.ModelInfo.Raw != nil {
		// 	litellmParams.ModelInfo = model.Spec.LiteLLMParams.ModelInfo
		// } else {
		// 	litellmParams.ModelInfo = litellm.NewModelInfo()
		// }

		// Handle MockResponse
		if model.Spec.LiteLLMParams.MockResponse != nil && *model.Spec.LiteLLMParams.MockResponse != "" {
			litellmParams.MockResponse = model.Spec.LiteLLMParams.MockResponse
		}

		// Handle AdditionalProps
		// if model.Spec.LiteLLMParams.AdditionalProps.Raw != nil {
		// 	litellmParams.AdditionalProperties = make(map[string]interface{})
		// 	// Convert AdditionalProps to map[string]interface{} if needed
		// 	for key, value := range model.Spec.LiteLLMParams.AdditionalProps.Raw {
		// 		litellmParams.AdditionalProperties[key] = value
		// 	}
		// }

		if err := parseAndAssign(model.Spec.LiteLLMParams.OutputCostPerToken, litellmParams.OutputCostPerToken, "OutputCostPerToken"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.OutputCostPerSecond, litellmParams.OutputCostPerSecond, "OutputCostPerSecond"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.OutputCostPerPixel, litellmParams.OutputCostPerPixel, "OutputCostPerPixel"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.InputCostPerPixel, litellmParams.InputCostPerPixel, "InputCostPerPixel"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.InputCostPerSecond, litellmParams.InputCostPerSecond, "InputCostPerSecond"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.InputCostPerToken, litellmParams.InputCostPerToken, "InputCostPerToken"); err != nil {
			return nil, err
		}
		if err := parseAndAssign(model.Spec.LiteLLMParams.MaxBudget, litellmParams.MaxBudget, "MaxBudget"); err != nil {
			return nil, err
		}
		modelRequest.LiteLLMParams = litellmParams
	}

	// Convert ModelInfo
	// if !reflect.DeepEqual(model.Spec.ModelInfo, litellmv1alpha1.ModelInfo{}) {
	// 	modelInfo := &litellm.ModelInfo{
	// 		ID:                  model.Spec.ModelInfo.ID,
	// 		DBModel:             model.Spec.ModelInfo.DBModel,
	// 		UpdatedBy:           model.Spec.ModelInfo.UpdatedBy,
	// 		CreatedBy:           model.Spec.ModelInfo.CreatedBy,
	// 		BaseModel:           model.Spec.ModelInfo.BaseModel,
	// 		Tier:                model.Spec.ModelInfo.Tier,
	// 		TeamID:              model.Spec.ModelInfo.TeamID,
	// 		TeamPublicModelName: model.Spec.ModelInfo.TeamPublicModelName,
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

func (r *ModelReconciler) updateConditions(ctx context.Context, model *litellmv1alpha1.Model, condition metav1.Condition) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if meta.SetStatusCondition(&model.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, model); err != nil {
			log.Error(err, "unable to update Model status with condition")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
