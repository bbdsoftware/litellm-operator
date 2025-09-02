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

package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	modelProvider "github.com/bbdsoftware/litellm-operator/internal/model"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	*base.BaseController[*litellmv1alpha1.Model]
	LitellmModelClient litellm.LitellmModel
}

// NewModelReconciler creates a new ModelReconciler instance
func NewModelReconciler(client client.Client, scheme *runtime.Scheme) *ModelReconciler {
	return &ModelReconciler{
		BaseController: base.NewBaseController[*litellmv1alpha1.Model](client, scheme),
	}
}

// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/finalizers,verbs=update

// ============================================================================
// Main Reconciler
// ============================================================================

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	// Phase 1: Fetch and validate the Model resource
	model := &litellmv1alpha1.Model{}
	model, err := r.FetchResource(ctx, req.NamespacedName, model)
	if err != nil {
		return ctrl.Result{}, err
	}
	if model == nil {
		// Model was deleted, no reconciliation needed
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling Model resource", "model", model.Name)

	// If a LitellmModelClient was injected (tests), reuse it; otherwise create one from connection details
	if r.LitellmModelClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, model.Spec.ConnectionRef, model.Namespace)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, model, err, base.ReasonConnectionError)
		}
		r.LitellmModelClient = litellmConnectionHandler.GetLitellmClient()
	}

	// Phase 3: Handle deletion if resource is being deleted
	if model.GetDeletionTimestamp() != nil {
		log.Info("Model resource is being deleted", "model", model.Name)
		return r.HandleDeletionWithFinalizer(ctx, model, util.FinalizerName, func(ctx context.Context, model *litellmv1alpha1.Model) error {
			return r.handleDeletion(ctx, model)
		})
	}

	// Phase 4: Handle model creation or update
	if r.shouldCreateModel(model) {
		if err := r.handleCreation(ctx, model); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, model, err, base.ReasonCreateFailed)
		}

		// Successfully created, requeue to handle the next phase
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Phase 5: Handle model updates (ModelId is present)
	if model.Status.ModelId == nil || *model.Status.ModelId == "" {
		log.Info("Model has no ModelId, skipping update check", "model", model.Name)
		return ctrl.Result{}, nil
	}

	// Convert model to ModelRequest for update operations
	modelRequest, err := r.convertToModelRequest(ctx, model)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, model, err, base.ReasonConversionFailed)
	}

	// ModelId present: attempt to fetch the remote model
	existingModel, err := r.LitellmModelClient.GetModelInfo(ctx, strings.TrimSpace(*model.Status.ModelId))
	if err != nil {
		log.Error(err, "Failed to get existing model from LiteLLM for model with id", "model", model.Name, "modelId", *model.Status.ModelId)
		r.HandleError(ctx, model, err, base.ReasonRemoteModelMissing)
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Remote model exists - check whether an update is required
	modelUpdateNeeded, err := r.LitellmModelClient.IsModelUpdateNeeded(ctx, &existingModel, modelRequest)
	if err != nil {
		log.Error(err, "Failed to check if model needs update")
		return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, model, err, base.ReasonUpdateFailed)
	}
	if modelUpdateNeeded.NeedsUpdate {
		log.Info("Model needs update, updating in LiteLLM")
		if err := r.handleUpdate(ctx, model, modelRequest); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.HandleError(ctx, model, err, base.ReasonUpdateFailed)
		}
	}

	// Model exists and is up-to-date - ensure conditions reflect healthy state
	if err := r.HandleSuccess(ctx, model, "Model is up-to-date"); err != nil {
		log.Error(err, "Failed to update conditions for up-to-date model")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	log.Info("Model exists and is up-to-date", "model", model.Name, "modelId", *model.Status.ModelId)
	return ctrl.Result{}, nil

}

// ============================================================================
// Resource Handler Functions
// ============================================================================

// shouldCreateModel determines if a new model should be created
func (r *ModelReconciler) shouldCreateModel(model *litellmv1alpha1.Model) bool {
	return model.Status.ModelId == nil || *model.Status.ModelId == ""
}

// handleCreation handles the creation of a new model
func (r *ModelReconciler) handleCreation(ctx context.Context, model *litellmv1alpha1.Model) error {

	log := logf.FromContext(ctx)

	r.SetProgressingConditions(model, "Creating model in LiteLLM")

	modelRequest, err := r.convertToModelRequest(ctx, model)
	if err != nil {
		r.SetProgressingConditions(model, "Creating model in LiteLLM")
		if updateErr := r.PatchStatus(ctx, model); updateErr != nil {
			return updateErr
		}
		return err
	}

	modelResponse, err := r.LitellmModelClient.CreateModel(ctx, modelRequest)
	if err != nil {
		r.SetErrorConditions(model, base.ReasonCreateFailed, err.Error())
		updateErr := r.PatchStatus(ctx, model)
		if updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
			// Return the original error, not the status update error
		}
		return err
	}

	// Populate status fields (including observed generation and last updated)
	updateModelStatus(model, &modelResponse)

	//update the crd status
	if err := r.HandleSuccess(ctx, model, "Model successfully created in LiteLLM"); err != nil {
		log.Error(err, "Failed to update conditions")
		return err
	}

	// Ensure finalizer is present after status is persisted
	if err := r.EnsureFinalizer(ctx, model, util.FinalizerName); err != nil {
		return err
	}

	log.Info("Successfully created model " + model.Name + " in LiteLLM")
	return nil
}

// handleUpdate handles the update of an existing model
func (r *ModelReconciler) handleUpdate(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) error {
	log := logf.FromContext(ctx)

	_, err := r.LitellmModelClient.UpdateModel(ctx, modelRequest)
	if err != nil {
		log.Error(err, "Failed to update model in LiteLLM", "modelName", model.Spec.ModelName)
		r.SetErrorConditions(model, base.ReasonUpdateFailed, err.Error())
		if updateErr := r.PatchStatus(ctx, model); updateErr != nil {
			return updateErr
		}
		return err
	}

	// Clear any error conditions and set ready
	if err := r.HandleSuccess(ctx, model, "Model successfully updated in LiteLLM"); err != nil {
		log.Error(err, "Failed to update conditions after successful update")
		return err
	}

	log.Info("Successfully updated model in LiteLLM", "modelName", model.Spec.ModelName)
	return nil
}

// handleDeletion handles the deletion of a model (called by HandleDeletionWithFinalizer)
func (r *ModelReconciler) handleDeletion(ctx context.Context, model *litellmv1alpha1.Model) error {
	log := logf.FromContext(ctx)

	// If there is a remote ModelId attempt to delete it; treat not-found as success
	if model.Status.ModelId != nil && *model.Status.ModelId != "" {
		err := r.LitellmModelClient.DeleteModel(ctx, *model.Status.ModelId)
		if err != nil {
			// If the remote model is already gone, proceed to cleanup
			if errors.Is(err, litellm.ErrNotFound) {
				log.Info("Remote model already not found in LiteLLM; proceeding to cleanup", "modelId", *model.Status.ModelId)
			} else {
				log.Error(err, "Failed to delete model from LiteLLM")
				return err
			}
		} else {
			log.Info("Successfully deleted model from LiteLLM", "modelId", *model.Status.ModelId)
		}
	} else {
		log.Info("No remote ModelId present; skipping remote delete")
	}

	// Clear ModelId and other status fields
	model.Status.ModelId = nil
	model.Status.ModelName = nil
	now := metav1.Now()
	model.Status.LastUpdated = &now
	if err := r.PatchStatus(ctx, model); err != nil {
		log.Error(err, "Failed to update Model status during deletion cleanup")
		return err
	}

	log.Info("Model resource cleanup complete")
	return nil
}

// ============================================================================
// Model Provider and Conversion Functions
// ============================================================================

// determine model provider from the Model
func (r *ModelReconciler) determineModelProvider(model *litellmv1alpha1.Model) (*modelProvider.ModelProvider, error) {
	if model.Spec.LiteLLMParams.CustomLLMProvider != nil {
		return modelProvider.NewModelProvider(*model.Spec.LiteLLMParams.CustomLLMProvider)
	}

	if model.Spec.LiteLLMParams.Model != nil {
		parts := strings.SplitN(*model.Spec.LiteLLMParams.Model, "/", 2)
		if len(parts) == 2 {
			return modelProvider.NewModelProvider(parts[0])
		}
	}

	return nil, fmt.Errorf("unable to determine model provider for model %s", model.Name)
}

// convertToModelRequest converts a Kubernetes Model to a LiteLLM ModelRequest
func (r *ModelReconciler) convertToModelRequest(ctx context.Context, model *litellmv1alpha1.Model) (*litellm.ModelRequest, error) {
	log := logf.FromContext(ctx)

	if model.Spec.LiteLLMParams.Model == nil {
		return nil, fmt.Errorf("LiteLLMParams.Model is not set")
	}

	llmProvider, err := r.determineModelProvider(model)
	if err != nil {
		return nil, err
	}

	// get the secret from the model secret ref and get the map of keys and values
	secretMap, err := util.GetMapFromSecret(ctx, r.Client, client.ObjectKey{
		Namespace: model.Spec.ModelSecretRef.Namespace,
		Name:      model.Spec.ModelSecretRef.SecretName,
	})
	if err != nil {
		return nil, err
	}

	err = llmProvider.ValidateConfig(secretMap)
	if err != nil {
		log.Error(err, "Failed to validate model provider config secret provided")
		return nil, err
	}

	// Append a short tag to indicate this model was created from the Model CRD
	modelRequest := &litellm.ModelRequest{
		ModelName: common.AppendModelSourceTag(model.Spec.ModelName, common.ModelTagCRD),
	}

	// Convert LiteLLMParams and map ApiKey from the secretMap if it exists
	if !reflect.DeepEqual(model.Spec.LiteLLMParams, litellmv1alpha1.LiteLLMParams{}) {

		// helper to safely convert optional pointer fields
		getStrFromPtr := func(s *string) *string {
			if s == nil || *s == "" {
				return nil
			}
			return util.StringPtrOrNil(*s)
		}
		getIntFromPtr := func(i *int) *int {
			if i == nil {
				return nil
			}
			return util.IntPtrOrNil(*i)
		}

		// convert an *int (from CRD) to *float64 expected by LiteLLM params (for timeout fields)
		getFloatFromIntPtr := func(i *int) *float64 {
			if i == nil {
				return nil
			}
			f := float64(*i)
			return &f
		}

		litellmParams := &litellm.UpdateLiteLLMParams{
			ApiKey:                   util.StringPtrOrNil(secretMap["apiKey"]),
			ApiBase:                  util.StringPtrOrNil(secretMap["apiBase"]),
			ApiVersion:               getStrFromPtr(model.Spec.LiteLLMParams.ApiVersion),
			VertexProject:            getStrFromPtr(model.Spec.LiteLLMParams.VertexProject),
			VertexLocation:           getStrFromPtr(model.Spec.LiteLLMParams.VertexLocation),
			RegionName:               getStrFromPtr(model.Spec.LiteLLMParams.RegionName),
			AwsAccessKeyID:           util.StringPtrOrNil(secretMap["AwsAccessKeyID"]),
			AwsSecretAccessKey:       util.StringPtrOrNil(secretMap["AwsSecretAccessKey"]),
			AwsRegionName:            getStrFromPtr(model.Spec.LiteLLMParams.AwsRegionName),
			WatsonXRegionName:        getStrFromPtr(model.Spec.LiteLLMParams.WatsonXRegionName),
			CustomLLMProvider:        getStrFromPtr(model.Spec.LiteLLMParams.CustomLLMProvider),
			TPM:                      getIntFromPtr(model.Spec.LiteLLMParams.TPM),
			RPM:                      getIntFromPtr(model.Spec.LiteLLMParams.RPM),
			MaxRetries:               getIntFromPtr(model.Spec.LiteLLMParams.MaxRetries),
			Organization:             getStrFromPtr(model.Spec.LiteLLMParams.Organization),
			LiteLLMCredentialName:    getStrFromPtr(model.Spec.LiteLLMParams.LiteLLMCredentialName),
			LiteLLMTraceID:           getStrFromPtr(model.Spec.LiteLLMParams.LiteLLMTraceID),
			MaxFileSizeMB:            getIntFromPtr(model.Spec.LiteLLMParams.MaxFileSizeMB),
			BudgetDuration:           getStrFromPtr(model.Spec.LiteLLMParams.BudgetDuration),
			UseInPassThrough:         model.Spec.LiteLLMParams.UseInPassThrough,
			UseLiteLLMProxy:          model.Spec.LiteLLMParams.UseLiteLLMProxy,
			AutoRouterConfigPath:     getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterConfigPath),
			AutoRouterConfig:         getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterConfig),
			AutoRouterDefaultModel:   getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterDefaultModel),
			AutoRouterEmbeddingModel: getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterEmbeddingModel),
			VertexCredentials:        util.StringPtrOrNil(secretMap["VertexCredentials"]),
			Timeout:                  getFloatFromIntPtr(model.Spec.LiteLLMParams.Timeout),
			StreamTimeout:            getFloatFromIntPtr(model.Spec.LiteLLMParams.StreamTimeout),
			MockResponse:             getStrFromPtr(model.Spec.LiteLLMParams.MockResponse),
			Model:                    getStrFromPtr(model.Spec.LiteLLMParams.Model),
		}

		// Handle ConfigurableClientsideAuthParams
		if model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams != nil && len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams) > 0 {
			litellmParams.ConfigurableClientsideAuthParams = make([]interface{}, len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams))
			for i, param := range *model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams {
				litellmParams.ConfigurableClientsideAuthParams[i] = param
			}
		}

		if model.Spec.LiteLLMParams.MergeReasoningContentInChoices != nil {
			litellmParams.MergeReasoningContentInChoices = model.Spec.LiteLLMParams.MergeReasoningContentInChoices
		} else {
			// default to false
			falseVal := false
			litellmParams.MergeReasoningContentInChoices = &falseVal
		}

		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.OutputCostPerToken, litellmParams.OutputCostPerToken, "OutputCostPerToken"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.OutputCostPerSecond, litellmParams.OutputCostPerSecond, "OutputCostPerSecond"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.OutputCostPerPixel, litellmParams.OutputCostPerPixel, "OutputCostPerPixel"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.InputCostPerPixel, litellmParams.InputCostPerPixel, "InputCostPerPixel"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.InputCostPerSecond, litellmParams.InputCostPerSecond, "InputCostPerSecond"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.InputCostPerToken, litellmParams.InputCostPerToken, "InputCostPerToken"); err != nil {
			return nil, err
		}
		if err := common.ParseAndAssign(model.Spec.LiteLLMParams.MaxBudget, litellmParams.MaxBudget, "MaxBudget"); err != nil {
			return nil, err
		}
		modelRequest.LiteLLMParams = litellmParams
		modelRequest.ModelInfo = litellm.NewModelInfo()
		if model.Status.ModelId != nil && *model.Status.ModelId != "" {
			modelRequest.ModelInfo.ID = model.Status.ModelId
		}
	}

	return modelRequest, nil
}

func updateModelStatus(model *litellmv1alpha1.Model, modelResponse *litellm.ModelResponse) {

	// Update observed generation and timestamp
	model.Status.ObservedGeneration = model.Generation
	now := metav1.Now()
	model.Status.LastUpdated = &now

	if modelResponse != nil {
		// ModelName from response
		if modelResponse.ModelName != "" {
			model.Status.ModelName = &modelResponse.ModelName
		} else {
			model.Status.ModelName = nil
		}

		// ModelId if present
		if modelResponse.ModelInfo != nil {
			model.Status.ModelId = modelResponse.ModelInfo.ID
		} else {
			model.Status.ModelId = nil
		}

		if modelResponse.LiteLLMParams != nil && modelResponse.LiteLLMParams != (&litellm.UpdateLiteLLMParams{}) {
			model.Status.LiteLLMParams = &litellmv1alpha1.LiteLLMParams{
				CustomLLMProvider: modelResponse.LiteLLMParams.CustomLLMProvider,
				Model:             modelResponse.LiteLLMParams.Model,
			}
		}
	} else {
		model.Status.ModelName = nil
		model.Status.ModelId = nil
	}

}

// ============================================================================
// Setup Functions
// ============================================================================

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&litellmv1alpha1.Model{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-model").
		Complete(r)
}
