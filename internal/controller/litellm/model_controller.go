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
	"fmt"
	"reflect"
	"strings"

	"errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	modelProvider "github.com/bbdsoftware/litellm-operator/internal/model"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	LitellmModelClient litellm.LitellmModel
	connectionHandler  *common.ConnectionHandler
}

// short tags appended to ModelName to indicate source
const (
	ModelTagCRD  = "-[crd]"
	ModelTagInst = "-[inst]"
)

// appendModelSourceTag appends a short tag to the provided modelName if not already present
func appendModelSourceTag(modelName string, tag string) string {
	if strings.HasSuffix(modelName, tag) {
		return modelName
	}
	return modelName + tag
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
		if kerrors.IsNotFound(err) {
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
		if updateErr := r.updateConditions(ctx, model, metav1.Condition{
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
	modelRequest, err := r.convertToModelRequest(ctx, model)
	if err != nil {
		log.Error(err, "Failed to convert Model to ModelRequest")
		if err := r.updateConditions(ctx, model, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// If there is no remote ModelId recorded, create the model in LiteLLM
	if model.Status.ModelId == nil || *model.Status.ModelId == "" {
		log.Info("No ModelId present; creating model in LiteLLM", "model", model.Name)
		return r.handleCreation(ctx, model, modelRequest)
	}

	// ModelId present: attempt to fetch the remote model
	existingModel, err := r.LitellmModelClient.GetModel(ctx, strings.TrimSpace(*model.Status.ModelId))
	if err != nil {
		// If remote model is not found, treat as an error and return (do not auto-create)
		if errors.Is(err, litellm.ErrNotFound) {
			log.Error(err, "Remote model ID not found in LiteLLM; refusing to auto-create to avoid duplicates", "modelId", *model.Status.ModelId)
			if updateErr := r.updateConditions(ctx, model, metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "RemoteModelMissing",
				Message:            "Remote model id not found in LiteLLM: " + err.Error(),
			}); updateErr != nil {
				log.Error(updateErr, "Failed to update conditions")
			}
			return ctrl.Result{}, err
		}

		// Other errors - surface and mark not ready
		log.Error(err, "Failed to get existing model from LiteLLM", "modelId", *model.Status.ModelId)
		if updateErr := r.updateConditions(ctx, model, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "UnableToCheckModelExists",
			Message:            err.Error(),
		}); updateErr != nil {
			log.Error(updateErr, "Failed to update conditions")
		}
		return ctrl.Result{}, err
	}

	// Remote model exists - check whether an update is required
	if r.LitellmModelClient.IsModelUpdateNeeded(ctx, &existingModel, modelRequest) {
		log.Info("Model needs update, updating in LiteLLM")
		return r.handleUpdate(ctx, model, modelRequest)
	}

	// No action required
	log.Info("Model exists and is up-to-date", "model", model.Name, "modelId", *model.Status.ModelId)
	return ctrl.Result{}, nil
}

// handleCreation handles the creation of a new model
func (r *ModelReconciler) handleCreation(ctx context.Context, model *litellmv1alpha1.Model, modelRequest *litellm.ModelRequest) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	modelResponse, err := r.LitellmModelClient.CreateModel(ctx, modelRequest)
	if err != nil {
		if err := r.updateConditions(ctx, model, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "LitellmError",
			Message: err.Error(),
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Populate status fields (including observed generation and last updated)
	updateModelStatus(model, &modelResponse)

	//update the crd status
	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update Model status")
		return ctrl.Result{}, err
	}

	// Persist status (ModelId, ModelName, ObservedGeneration, LastUpdated) along with the Ready condition
	if err := r.updateConditions(ctx, model, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "LitellmSuccess",
		Message:            "Model created in Litellm",
		LastTransitionTime: metav1.Now(),
	}); err != nil {
		log.Error(err, "Failed to update Model status with Ready condition")
		return ctrl.Result{}, err
	}

	// Ensure finalizer is present after status is persisted
	if !controllerutil.ContainsFinalizer(model, util.FinalizerName) {
		controllerutil.AddFinalizer(model, util.FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	log.Info("Successfully created model " + model.Name + " in LiteLLM")
	return ctrl.Result{}, nil
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

	// If there is a remote ModelId attempt to delete it; treat not-found as success
	if model.Status.ModelId != nil && *model.Status.ModelId != "" {
		err := r.LitellmModelClient.DeleteModel(ctx, *model.Status.ModelId)
		if err != nil {
			// If the remote model is already gone, proceed to remove finalizer
			if errors.Is(err, litellm.ErrNotFound) {
				log.Info("Remote model already not found in LiteLLM; proceeding to cleanup", "modelId", *model.Status.ModelId)
			} else {
				log.Error(err, "Failed to delete model from LiteLLM")
				// Update condition to surface the deletion failure
				if updateErr := r.updateConditions(ctx, model, metav1.Condition{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             "DeleteFailed",
					Message:            err.Error(),
				}); updateErr != nil {
					log.Error(updateErr, "Failed to update conditions after deletion failure")
				}
				return ctrl.Result{}, err
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
	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update Model status during deletion cleanup")
		return ctrl.Result{}, err
	}

	// Remove finalizer and persist
	if controllerutil.ContainsFinalizer(model, util.FinalizerName) {
		controllerutil.RemoveFinalizer(model, util.FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to remove finalizer from Model")
			return ctrl.Result{}, err
		}
	}

	log.Info("Model resource cleanup complete; finalizer removed")
	return ctrl.Result{}, nil
}

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
		ModelName: appendModelSourceTag(model.Spec.ModelName, ModelTagCRD),
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

		litellmParams := &litellm.UpdateLiteLLMParams{
			ApiKey:                         util.StringPtrOrNil(secretMap["apiKey"]),
			ApiBase:                        util.StringPtrOrNil(secretMap["apiBase"]),
			ApiVersion:                     getStrFromPtr(model.Spec.LiteLLMParams.ApiVersion),
			VertexProject:                  getStrFromPtr(model.Spec.LiteLLMParams.VertexProject),
			VertexLocation:                 getStrFromPtr(model.Spec.LiteLLMParams.VertexLocation),
			RegionName:                     getStrFromPtr(model.Spec.LiteLLMParams.RegionName),
			AwsAccessKeyID:                 util.StringPtrOrNil(secretMap["AwsAccessKeyID"]),
			AwsSecretAccessKey:             util.StringPtrOrNil(secretMap["AwsSecretAccessKey"]),
			AwsRegionName:                  getStrFromPtr(model.Spec.LiteLLMParams.AwsRegionName),
			WatsonXRegionName:              getStrFromPtr(model.Spec.LiteLLMParams.WatsonXRegionName),
			CustomLLMProvider:              getStrFromPtr(model.Spec.LiteLLMParams.CustomLLMProvider),
			TPM:                            getIntFromPtr(model.Spec.LiteLLMParams.TPM),
			RPM:                            getIntFromPtr(model.Spec.LiteLLMParams.RPM),
			MaxRetries:                     getIntFromPtr(model.Spec.LiteLLMParams.MaxRetries),
			Organization:                   getStrFromPtr(model.Spec.LiteLLMParams.Organization),
			LiteLLMCredentialName:          getStrFromPtr(model.Spec.LiteLLMParams.LiteLLMCredentialName),
			LiteLLMTraceID:                 getStrFromPtr(model.Spec.LiteLLMParams.LiteLLMTraceID),
			MaxFileSizeMB:                  getIntFromPtr(model.Spec.LiteLLMParams.MaxFileSizeMB),
			BudgetDuration:                 getStrFromPtr(model.Spec.LiteLLMParams.BudgetDuration),
			UseInPassThrough:               model.Spec.LiteLLMParams.UseInPassThrough,
			UseLiteLLMProxy:                model.Spec.LiteLLMParams.UseLiteLLMProxy,
			MergeReasoningContentInChoices: model.Spec.LiteLLMParams.MergeReasoningContentInChoices,
			AutoRouterConfigPath:           getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterConfigPath),
			AutoRouterConfig:               getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterConfig),
			AutoRouterDefaultModel:         getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterDefaultModel),
			AutoRouterEmbeddingModel:       getStrFromPtr(model.Spec.LiteLLMParams.AutoRouterEmbeddingModel),
			VertexCredentials:              util.StringPtrOrNil(secretMap["VertexCredentials"]),
			Timeout:                        getIntFromPtr(model.Spec.LiteLLMParams.Timeout),
			StreamTimeout:                  getIntFromPtr(model.Spec.LiteLLMParams.StreamTimeout),
			MockResponse:                   getStrFromPtr(model.Spec.LiteLLMParams.MockResponse),
			Model:                          getStrFromPtr(model.Spec.LiteLLMParams.Model),
		}

		// Handle ConfigurableClientsideAuthParams
		if model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams != nil && len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams) > 0 {
			litellmParams.ConfigurableClientsideAuthParams = make([]interface{}, len(*model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams))
			for i, param := range *model.Spec.LiteLLMParams.ConfigurableClientsideAuthParams {
				litellmParams.ConfigurableClientsideAuthParams[i] = param
			}
		}

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

	return modelRequest, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&litellmv1alpha1.Model{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("litellm-model").
		Complete(r)
}

func (r *ModelReconciler) updateConditions(ctx context.Context, model *litellmv1alpha1.Model, condition metav1.Condition) error {
	log := logf.FromContext(ctx)

	if meta.SetStatusCondition(&model.Status.Conditions, condition) {
		if err := r.Status().Update(ctx, model); err != nil {
			log.Error(err, "unable to update Model status with condition")
			return err
		}
	}

	return nil
}
