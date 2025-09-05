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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	modelProvider "github.com/bbdsoftware/litellm-operator/internal/model"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	*base.BaseController[*litellmv1alpha1.Model]
	LitellmModelClient litellm.LitellmModel
}

type ExternalData struct {
	ModelID   string `json:"modelID"`
	ModelName string `json:"modelName"`
}

// NewModelReconciler creates a new ModelReconciler instance
func NewModelReconciler(client client.Client, scheme *runtime.Scheme) *ModelReconciler {
	return &ModelReconciler{
		BaseController: &base.BaseController[*litellmv1alpha1.Model]{
			Client:         client,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
		},
		LitellmModelClient: nil,
	}
}

// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=models/finalizers,verbs=update

// ============================================================================
// Main Reconciler
// ============================================================================

// Reconcile implements the single-loop ensure* pattern with finalizer, conditions, and drift sync
func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Add timeout to avoid long-running reconciliation
	ctx, cancel := r.WithTimeout(ctx)
	defer cancel()

	log := logf.FromContext(ctx)

	// Phase 1: Fetch and validate the resource
	model := &litellmv1alpha1.Model{}
	model, err := r.FetchResource(ctx, req.NamespacedName, model)
	if err != nil {
		log.Error(err, "Failed to get Model")
		return ctrl.Result{}, err
	}
	if model == nil {
		return ctrl.Result{}, nil
	}

	// Phase 2: Set up connections and clients
	if err := r.ensureConnectionSetup(ctx, model); err != nil {
		log.Error(err, "Failed to setup connections")
		return r.HandleErrorRetryable(ctx, model, err, base.ReasonConnectionError)
	}

	// Phase 3: Handle deletion if resource is being deleted
	if !model.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, model)
	}

	// Phase 4: Upsert branch - ensure finalizer
	if err := r.AddFinalizer(ctx, model, util.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	var externalData ExternalData
	// Phase 5: Ensure external resource (create/patch/repair drift)
	if res, err := r.ensureExternal(ctx, model, &externalData); res.Requeue || err != nil {
		return res, err
	}

	// Phase 6: Ensure in-cluster children (owned -> GC on delete)
	if err := r.ensureChildren(ctx, model, &externalData); err != nil {
		return r.HandleCommonErrors(ctx, model, err)
	}

	// Phase 7: Mark Ready and persist ObservedGeneration
	r.SetSuccessConditions(model, "Model is in desired state")
	model.Status.ObservedGeneration = model.GetGeneration()
	if err := r.PatchStatus(ctx, model); err != nil {
		return ctrl.Result{}, err
	}

	// Phase 8: Periodic drift sync (external might change out of band)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// ensureConnectionSetup configures the LiteLLM client
func (r *ModelReconciler) ensureConnectionSetup(ctx context.Context, model *litellmv1alpha1.Model) error {
	if r.LitellmModelClient == nil {
		litellmConnectionHandler, err := common.NewLitellmConnectionHandler(r.Client, ctx, model.Spec.ConnectionRef, model.Namespace)
		if err != nil {
			return err
		}
		r.LitellmModelClient = litellmConnectionHandler.GetLitellmClient()
	}

	return nil
}

// reconcileDelete handles the deletion branch with idempotent external cleanup
func (r *ModelReconciler) reconcileDelete(ctx context.Context, model *litellmv1alpha1.Model) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !r.HasFinalizer(model, util.FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Set deleting condition and update status
	r.SetCondition(model, base.CondReady, metav1.ConditionFalse, "Deleting", "Model is being deleted")
	if err := r.PatchStatus(ctx, model); err != nil {
		log.Error(err, "Failed to update status during deletion")
		// Continue with deletion even if status update fails
	}

	// Idempotent external cleanup
	if model.Status.ModelId != nil && *model.Status.ModelId != "" {
		err := r.LitellmModelClient.DeleteModel(ctx, *model.Status.ModelId)
		if err != nil {
			// If the remote model is already gone, proceed to cleanup
			if errors.Is(err, litellm.ErrNotFound) {
				log.Info("Remote model already not found in LiteLLM; proceeding to cleanup", "modelId", *model.Status.ModelId)
			} else {
				log.Error(err, "Failed to delete model from LiteLLM")
				return r.HandleErrorRetryable(ctx, model, err, base.ReasonDeleteFailed)
			}
		} else {
			log.Info("Successfully deleted model from LiteLLM", "modelId", *model.Status.ModelId)
		}
	}

	// Remove finalizer
	if err := r.RemoveFinalizer(ctx, model, util.FinalizerName); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return r.HandleErrorRetryable(ctx, model, err, base.ReasonDeleteFailed)
	}

	log.Info("Successfully deleted model", "model", model.Name)
	return ctrl.Result{}, nil
}

// ensureExternal manages the external model resource (create/patch/repair drift)
func (r *ModelReconciler) ensureExternal(ctx context.Context, model *litellmv1alpha1.Model, externalData *ExternalData) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Ensuring external model resource", "model", model.Name)

	// Set progressing condition
	r.SetProgressingConditions(model, "Reconciling model in LiteLLM")
	if err := r.PatchStatus(ctx, model); err != nil {
		log.Error(err, "Failed to update progressing status")
		// Continue despite status update failure
	}

	modelRequest, err := r.convertToModelRequest(ctx, model)
	if err != nil {
		log.Error(err, "Failed to create model request")
		return r.HandleErrorRetryable(ctx, model, err, base.ReasonInvalidSpec)
	}

	// Create if no external ID exists
	if model.Status.ModelId == nil || *model.Status.ModelId == "" {
		log.Info("Creating new model in LiteLLM", "modelName", model.Spec.ModelName)
		modelResponse, err := r.LitellmModelClient.CreateModel(ctx, modelRequest)
		if err != nil {
			log.Error(err, "Failed to create model in LiteLLM")
			return r.HandleErrorRetryable(ctx, model, err, base.ReasonLitellmError)
		}

		// Populate external data
		if modelResponse.ModelInfo != nil && modelResponse.ModelInfo.ID != nil {
			externalData.ModelID = *modelResponse.ModelInfo.ID
		}
		externalData.ModelName = modelResponse.ModelName

		r.updateModelStatus(model, &modelResponse)
		if err := r.PatchStatus(ctx, model); err != nil {
			log.Error(err, "Failed to update status after creation")
			return r.HandleErrorRetryable(ctx, model, err, base.ReasonReconcileError)
		}
		log.Info("Successfully created model in LiteLLM", "modelID", externalData.ModelID)
		return ctrl.Result{}, nil
	}

	// Model exists, check for drift and repair if needed
	log.V(1).Info("Checking for drift", "modelID", *model.Status.ModelId)
	observedModel, err := r.LitellmModelClient.GetModelInfo(ctx, strings.TrimSpace(*model.Status.ModelId))
	if err != nil {
		log.Error(err, "Failed to get model from LiteLLM")
		return r.HandleErrorRetryable(ctx, model, err, base.ReasonLitellmError)
	}

	updateNeeded, err := r.LitellmModelClient.IsModelUpdateNeeded(ctx, &observedModel, modelRequest)
	if err != nil {
		log.Error(err, "Failed to check if model needs update")
		return r.HandleErrorRetryable(ctx, model, err, base.ReasonLitellmError)
	}

	if updateNeeded.NeedsUpdate {
		log.Info("Repairing drift in LiteLLM", "modelName", model.Spec.ModelName, "changedFields", updateNeeded.ChangedFields)
		modelResponse, err := r.LitellmModelClient.UpdateModel(ctx, modelRequest)
		if err != nil {
			log.Error(err, "Failed to update model in LiteLLM")
			return r.HandleErrorRetryable(ctx, model, err, base.ReasonLitellmError)
		}

		// Populate external data
		if modelResponse.ModelInfo != nil && modelResponse.ModelInfo.ID != nil {
			externalData.ModelID = *modelResponse.ModelInfo.ID
		}
		externalData.ModelName = modelResponse.ModelName

		r.updateModelStatus(model, &modelResponse)
		if err := r.PatchStatus(ctx, model); err != nil {
			log.Error(err, "Failed to update status after update")
			return r.HandleErrorRetryable(ctx, model, err, base.ReasonReconcileError)
		}
		log.Info("Successfully repaired drift in LiteLLM", "modelID", *model.Status.ModelId)
	} else {
		log.V(1).Info("Model is up to date in LiteLLM", "modelID", *model.Status.ModelId)
	}

	return ctrl.Result{}, nil
}

// ensureChildren manages in-cluster child resources (models don't have children currently)
func (r *ModelReconciler) ensureChildren(ctx context.Context, model *litellmv1alpha1.Model, externalData *ExternalData) error {
	// Models don't currently have child resources, but this follows the pattern
	return nil
}

// updateModelStatus updates the status of the k8s Model from the litellm response
func (r *ModelReconciler) updateModelStatus(model *litellmv1alpha1.Model, modelResponse *litellm.ModelResponse) {
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
