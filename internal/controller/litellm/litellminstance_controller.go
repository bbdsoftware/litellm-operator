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

// Package litellm provides the controller implementation for LiteLLMInstance resources.
// This package contains the reconciliation logic for managing LiteLLM proxy instances
// in Kubernetes, including the creation and management of ConfigMaps, Secrets,
// Deployments, Services, and Ingress resources.
package litellm

import (
	"context"
	"errors"
	"errors"
	"fmt"
	"strconv"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"regexp"
	"strings"
	"regexp"
	"strings"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// FinalizerName is the name of the finalizer added to LiteLLMInstance
	// resources to ensure proper cleanup when the resource is deleted.
	FinalizerName = "litellm.litellm.ai/finalizer"

	// Container configuration constants
	ContainerPort = 4000                                    // Port that the LiteLLM container listens on
	ServicePort   = 80                                      // Port that the Service exposes
	ConfigPath    = "/etc/litellm/proxy_server_config.yaml" // Path to config file in container

	// Health check paths for container probes
	LivenessPath  = "/health/liveness"  // Path for liveness probe
	ReadinessPath = "/health/readiness" // Path for readiness probe
)

// Helper functions for descriptive return values
func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func RequeueAfter(duration time.Duration) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: duration}, nil
}

// LiteLLMInstanceReconciler reconciles a LiteLLMInstance object.
// It is responsible for creating and managing the Kubernetes resources required
// to run a LiteLLM proxy instance, including ConfigMaps, Secrets, Deployments,
// Services, and optionally Ingress resources.
type LiteLLMInstanceReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	litellmResourceNaming *util.LitellmResourceNaming
}

type LiteLLMParamsYAML struct {
	ApiKey                           string                 `yaml:"api_key,omitempty"`
	ApiBase                          string                 `yaml:"api_base,omitempty"`
	AwsAccessKeyID                   string                 `yaml:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey               string                 `yaml:"aws_secret_access_key,omitempty"`
	AwsRegionName                    string                 `yaml:"aws_region_name,omitempty"`
	AutoRouterConfigPath             string                 `yaml:"auto_router_config_path,omitempty"`
	AutoRouterConfig                 string                 `yaml:"auto_router_config,omitempty"`
	AutoRouterDefaultModel           string                 `yaml:"auto_router_default_model,omitempty"`
	AutoRouterEmbeddingModel         string                 `yaml:"auto_router_embedding_model,omitempty"`
	AdditionalProps                  map[string]interface{} `yaml:"additionalProps,omitempty"`
	ApiVersion                       string                 `yaml:"api_version,omitempty"`
	BudgetDuration                   string                 `yaml:"budget_duration,omitempty"`
	ConfigurableClientsideAuthParams []interface{}          `yaml:"configurable_clientside_auth_params,omitempty"`
	CustomLLMProvider                string                 `yaml:"custom_llm_provider,omitempty"`
	InputCostPerToken                float64                `yaml:"input_cost_per_token,omitempty"`
	InputCostPerPixel                float64                `yaml:"input_cost_per_pixel,omitempty"`
	InputCostPerSecond               float64                `yaml:"input_cost_per_second,omitempty"`
	LiteLLMTraceID                   string                 `yaml:"litellm_trace_id,omitempty"`
	LiteLLMCredentialName            string                 `yaml:"litellm_credential_name,omitempty"`
	MaxFileSizeMB                    int                    `yaml:"max_file_size_mb,omitempty"`
	MergeReasoningContentInChoices   bool                   `yaml:"merge_reasoning_content_in_choices,omitempty"`
	MockResponse                     string                 `yaml:"mock_response,omitempty"`
	Model                            string                 `yaml:"model"`
	MaxBudget                        float64                `yaml:"max_budget,omitempty"`
	MaxRetries                       int                    `yaml:"max_retries,omitempty"`
	Organization                     string                 `yaml:"organization,omitempty"`
	OutputCostPerToken               float64                `yaml:"output_cost_per_token,omitempty"`
	OutputCostPerSecond              float64                `yaml:"output_cost_per_second,omitempty"`
	OutputCostPerPixel               float64                `yaml:"output_cost_per_pixel,omitempty"`
	RegionName                       string                 `yaml:"region_name,omitempty"`
	RPM                              int                    `yaml:"rpm,omitempty"`
	StreamTimeout                    int                    `yaml:"stream_timeout,omitempty"`
	TPM                              int                    `yaml:"tpm,omitempty"`
	Timeout                          int                    `yaml:"timeout,omitempty"`
	UseInPassThrough                 bool                   `yaml:"use_in_pass_through,omitempty"`
	UseLiteLLMProxy                  bool                   `yaml:"use_litellm_proxy,omitempty"`
	VertexProject                    string                 `yaml:"vertex_project,omitempty"`
	VertexLocation                   string                 `yaml:"vertex_location,omitempty"`
	VertexCredentials                string                 `yaml:"vertex_credentials,omitempty"`
	WatsonxRegionName                string                 `yaml:"watsonx_region_name,omitempty"`
}

type ModelListItemYAML struct {
	ModelName     string            `yaml:"model_name"`
	LitellmParams LiteLLMParamsYAML `yaml:"litellm_params"`
}

type RouterSettingsYAML struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password,omitempty"`
}

type ProxyConfig struct {
	ModelList       []ModelListItemYAML `yaml:"model_list"`
	RouterSettings  RouterSettingsYAML  `yaml:"router_settings,omitempty"`
	GeneralSettings struct {
		AllowRequestsOnDBUnavailable bool `yaml:"allow_requests_on_db_unavailable"`
		StoreModelInDB               bool `yaml:"store_model_in_db"` //Needed to be able to store new models created via REST API
	} `yaml:"general_settings"`
}

// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=litellminstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=litellminstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=litellm.litellm.ai,resources=litellminstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the current state of the cluster closer to the desired state.
// It creates or updates all required Kubernetes resources for a LiteLLMInstance:
// ConfigMap, Secret, Deployment, Service, and optionally Ingress.
func (r *LiteLLMInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the LiteLLMInstance
	llm := &litellmv1alpha1.LiteLLMInstance{}
	if err := r.Get(ctx, req.NamespacedName, llm); err != nil {
		return RequeueWithError(client.IgnoreNotFound(err))
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(llm.Name)
	}

	if r.litellmResourceNaming == nil {
		r.litellmResourceNaming = util.NewLitellmResourceNaming(llm.Name)
	}

	// Check if the instance is being deleted
	if llm.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, llm)
	}

	// Add finalizer if it doesn't exist
	if !util.ContainsString(llm.Finalizers, FinalizerName) {
		llm.Finalizers = append(llm.Finalizers, FinalizerName)
		if err := r.Update(ctx, llm); err != nil {
			return RequeueWithError(err)
		}
	}

	_, err := validateModelListForDuplicates(llm)
	if err != nil {
		log.Error(err, "LLM Instance modelList is not valid")
		return ctrl.Result{}, err
	}

	_, err := validateModelListForDuplicates(llm)
	if err != nil {
		log.Error(err, "LLM Instance modelList is not valid")
		return ctrl.Result{}, err
	}

	// Create or update resources
	configMap, err := r.createConfigMap(ctx, llm)
	if err != nil {
		log.Error(err, "Failed to create or update ConfigMap")
		return util.HandleConflictError(err)
	}

	secret, err := r.createMasterKeySecret(ctx, llm)

	secret, err := r.createMasterKeySecret(ctx, llm)

	if err != nil {
		log.Error(err, "Failed to create or update Secret")
		return util.HandleConflictError(err)
	}

	// Create ServiceAccount for the LiteLLM instance
	serviceAccount, err := r.createServiceAccount(ctx, llm)
	if err != nil {
		log.Error(err, "Failed to create or update ServiceAccount")
		return util.HandleConflictError(err)
	}

	// Create Role and RoleBinding for the ServiceAccount
	if err := r.createRBAC(ctx, llm, serviceAccount); err != nil {
		log.Error(err, "Failed to create RBAC resources")
		return util.HandleConflictError(err)
	}

	deployment, err := r.createDeployment(ctx, llm, configMap, secret, serviceAccount)
	if err != nil {
		log.Error(err, "Failed to create or update Deployment")
		return util.HandleConflictError(err)
	}

	service, err := r.createService(ctx, llm)
	if err != nil {
		log.Error(err, "Failed to create or update Service")
		return util.HandleConflictError(err)
	}

	if llm.Spec.Ingress.Enabled {
		if err := r.createIngress(ctx, llm); err != nil {
			log.Error(err, "Failed to create or update Ingress")
			return util.HandleConflictError(err)
		}
	}

	// Update status to reflect successful reconciliation
	allReady, err := r.updateStatus(ctx, llm, deployment, service)
	if err != nil {
		log.Error(err, "Failed to update LiteLLMInstance status")
		return RequeueWithError(err)
	}

	// Check if all resources are ready, if not, requeue after a short delay
	if !allReady {
		log.Info("Resources not ready yet, requeuing", "name", llm.Name)
		return RequeueAfter(10 * time.Second)
	}

	log.Info("Successfully reconciled LiteLLMInstance", "name", llm.Name, "namespace", llm.Namespace)
	return DoNotRequeue()
}

// handleDeletion removes the finalizer to allow the resource to be fully deleted.
func (r *LiteLLMInstanceReconciler) handleDeletion(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Remove finalizer if it exists
	if util.ContainsString(llm.Finalizers, FinalizerName) {
		llm.Finalizers = util.RemoveString(llm.Finalizers, FinalizerName)
		if err := r.Update(ctx, llm); err != nil {
			return RequeueWithError(err)
		}
	}

	log.Info("LiteLLMInstance deletion completed", "name", llm.Name, "namespace", llm.Namespace)
	return DoNotRequeue()
}

func validateModelListForDuplicates(llm *litellmv1alpha1.LiteLLMInstance) (bool, error) {
	seen := make(map[string]bool)

	for _, model := range llm.Spec.Models {
		if seen[model.Identifier] {
			err := fmt.Errorf("LLM Instance model list contains duplicate identifiers %s", model.Identifier)
			return false, err
		}
		seen[model.Identifier] = true
	}

	return true, nil
}

func parseAndAssign(field *string, target *float64, fieldName string) error {
	if field != nil && *field != "" {
		value, err := strconv.ParseFloat(*field, 64)
		if err != nil {
			return errors.New(fieldName + " not parsable to float")
		}
		target := new(float64)
		*target = value
	}
	return nil
}

func validateModelListForDuplicates(llm *litellmv1alpha1.LiteLLMInstance) (bool, error) {
	seen := make(map[string]bool)

	for _, model := range llm.Spec.Models {
		if seen[model.Identifier] {
			err := fmt.Errorf("LLM Instance model list contains duplicate identifiers %s", model.Identifier)
			return false, err
		}
		seen[model.Identifier] = true
	}

	return true, nil
}

func parseAndAssign(field string, target *float64, fieldName string) error {
	if field != "" {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return errors.New(fieldName + " not parsable to float")
		}
		*target = value
	}
	return nil
}

// renderProxyConfig generates the YAML configuration for the LiteLLM proxy server.
// It creates a configuration structure with model list, router settings, and general settings.
func renderProxyConfig(llm *litellmv1alpha1.LiteLLMInstance, ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme) (string, error) {
	log := logf.FromContext(ctx)

	var modelListYAML []ModelListItemYAML
	if llm.Spec.Models != nil {
		for _, model := range llm.Spec.Models {

			if model.LiteLLMParams.Model == nil || *model.LiteLLMParams.Model == "" {
				err := fmt.Errorf("model name is required for each model in the list")
				log.Error(err, "Failed to render proxy config")
				return "", err
			}

			//map all LiteLLMParams to the YAML struct
			litellmParams := LiteLLMParamsYAML{
				ApiKey:                         util.DerefString(model.LiteLLMParams.ApiKey),
				ApiBase:                        util.DerefString(model.LiteLLMParams.ApiBase),
				AwsAccessKeyID:                 util.DerefString(model.LiteLLMParams.AwsAccessKeyID),
				AwsSecretAccessKey:             util.DerefString(model.LiteLLMParams.AwsSecretAccessKey),
				AwsRegionName:                  util.DerefString(model.LiteLLMParams.AwsRegionName),
				AutoRouterConfigPath:           util.DerefString(model.LiteLLMParams.AutoRouterConfigPath),
				AutoRouterConfig:               util.DerefString(model.LiteLLMParams.AutoRouterConfig),
				AutoRouterDefaultModel:         util.DerefString(model.LiteLLMParams.AutoRouterDefaultModel),
				AutoRouterEmbeddingModel:       util.DerefString(model.LiteLLMParams.AutoRouterEmbeddingModel),
				ApiVersion:                     util.DerefString(model.LiteLLMParams.ApiVersion),
				BudgetDuration:                 util.DerefString(model.LiteLLMParams.BudgetDuration),
				CustomLLMProvider:              util.DerefString(model.LiteLLMParams.CustomLLMProvider),
				LiteLLMTraceID:                 util.DerefString(model.LiteLLMParams.LiteLLMTraceID),
				LiteLLMCredentialName:          util.DerefString(model.LiteLLMParams.LiteLLMCredentialName),
				MergeReasoningContentInChoices: util.DerefBool(model.LiteLLMParams.MergeReasoningContentInChoices),
				MockResponse:                   util.DerefString(model.LiteLLMParams.MockResponse),
				Model:                          util.DerefString(model.LiteLLMParams.Model),
				MaxRetries:                     util.DerefInt(model.LiteLLMParams.MaxRetries),
				MaxFileSizeMB:                  util.DerefInt(model.LiteLLMParams.MaxFileSizeMB),
				Organization:                   util.DerefString(model.LiteLLMParams.Organization),
				RegionName:                     util.DerefString(model.LiteLLMParams.RegionName),
				RPM:                            util.DerefInt(model.LiteLLMParams.RPM),
				StreamTimeout:                  util.DerefInt(model.LiteLLMParams.StreamTimeout),
				TPM:                            util.DerefInt(model.LiteLLMParams.TPM),
				Timeout:                        util.DerefInt(model.LiteLLMParams.Timeout),
				UseInPassThrough:               util.DerefBool(model.LiteLLMParams.UseInPassThrough),
				UseLiteLLMProxy:                util.DerefBool(model.LiteLLMParams.UseLiteLLMProxy),
				VertexProject:                  util.DerefString(model.LiteLLMParams.VertexProject),
				VertexLocation:                 util.DerefString(model.LiteLLMParams.VertexLocation),
				VertexCredentials:              util.DerefString(model.LiteLLMParams.VertexCredentials),
				WatsonxRegionName:              util.DerefString(model.LiteLLMParams.WatsonXRegionName),
			}

			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerToken, &litellmParams.OutputCostPerToken, "OutputCostPerToken"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerSecond, &litellmParams.OutputCostPerSecond, "OutputCostPerSecond"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerPixel, &litellmParams.OutputCostPerPixel, "OutputCostPerPixel"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerPixel, &litellmParams.InputCostPerPixel, "InputCostPerPixel"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerSecond, &litellmParams.InputCostPerSecond, "InputCostPerSecond"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerToken, &litellmParams.InputCostPerSecond, "InputCostPerToken"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.MaxBudget, &litellmParams.MaxBudget, "MaxBudget"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}

			if model.LiteLLMParams.ConfigurableClientsideAuthParams != nil && len(*model.LiteLLMParams.ConfigurableClientsideAuthParams) > 0 {
				litellmParams.ConfigurableClientsideAuthParams = make([]interface{}, len(*model.LiteLLMParams.ConfigurableClientsideAuthParams))
				for i, param := range *model.LiteLLMParams.ConfigurableClientsideAuthParams {
					litellmParams.ConfigurableClientsideAuthParams[i] = param
				}
			}

			// If the model requires authentication, create individual secrets for each model
			if model.RequiresAuth {
				//get existing modelCredentials secret
				modelCredentials := &corev1.Secret{}
				err := k8sClient.Get(ctx, client.ObjectKey{Name: model.ModelCredentials.NameRef, Namespace: llm.Namespace}, modelCredentials)
				if err != nil {
					log.Error(err, "Model Credentials not found", "modelIdentifier", model.Identifier)
					return "", nil
				}

				//create individual secrets from the ModelCredentials
				secretNames, err := createAdditionalModelSecrets(ctx, &model, modelCredentials, llm, k8sClient, scheme)
				if err != nil {
					log.Error(err, "Failed to create secrets", "modelIdentifier", model.Identifier)
				}

				secretPrefix := "os.environ/"
				keys := model.ModelCredentials.Keys
				//match the secret to its yaml config to populate the yaml template
				fieldMap := map[string]*string{
					keys.VertexCredentials:  &litellmParams.VertexCredentials,
					keys.ApiBase:            &litellmParams.ApiBase,
					keys.AwsAccessKeyID:     &litellmParams.AwsAccessKeyID,
					keys.AwsSecretAccessKey: &litellmParams.AwsSecretAccessKey,
					keys.VertexProject:      &litellmParams.VertexCredentials,
					keys.ApiKey:             &litellmParams.ApiKey,
				}

				for key, target := range fieldMap {
					if key != "" {
						if secretName, ok := secretNames[key]; ok {
							*target = secretPrefix + secretName
						}
					}
				}

			}

			modelYAML := ModelListItemYAML{
				ModelName:     model.ModelName,
				LitellmParams: litellmParams,
			}
			modelListYAML = append(modelListYAML, modelYAML)
		}
	}
func renderProxyConfig(llm *litellmv1alpha1.LiteLLMInstance, ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme) (string, error) {
	log := logf.FromContext(ctx)

	var modelListYAML []ModelListItemYAML
	if llm.Spec.Models != nil {
		for _, model := range llm.Spec.Models {

			if model.LiteLLMParams.Model == "" {
				err := fmt.Errorf("model name is required for each model in the list")
				log.Error(err, "Failed to render proxy config")
				return "", err
			}

			//map all LiteLLMParams to the YAML struct
			litellmParams := LiteLLMParamsYAML{
				ApiKey:                           model.LiteLLMParams.ApiKey,
				ApiBase:                          model.LiteLLMParams.ApiBase,
				AwsAccessKeyID:                   model.LiteLLMParams.AwsAccessKeyID,
				AwsSecretAccessKey:               model.LiteLLMParams.AwsSecretAccessKey,
				AwsRegionName:                    model.LiteLLMParams.AwsRegionName,
				AutoRouterConfigPath:             model.LiteLLMParams.AutoRouterConfigPath,
				AutoRouterConfig:                 model.LiteLLMParams.AutoRouterConfig,
				AutoRouterDefaultModel:           model.LiteLLMParams.AutoRouterDefaultModel,
				AutoRouterEmbeddingModel:         model.LiteLLMParams.AutoRouterEmbeddingModel,
				AdditionalProps:                  model.LiteLLMParams.AdditionalProps,
				ApiVersion:                       model.LiteLLMParams.ApiVersion,
				BudgetDuration:                   model.LiteLLMParams.BudgetDuration,
				ConfigurableClientsideAuthParams: model.LiteLLMParams.ConfigurableClientsideAuthParams,
				CustomLLMProvider:                model.LiteLLMParams.CustomLLMProvider,
				LiteLLMTraceID:                   model.LiteLLMParams.LiteLLMTraceID,
				LiteLLMCredentialName:            model.LiteLLMParams.LiteLLMCredentialName,
				MergeReasoningContentInChoices:   model.LiteLLMParams.MergeReasoningContentInChoices,
				MockResponse:                     model.LiteLLMParams.MockResponse,
				Model:                            model.LiteLLMParams.Model,
				MaxRetries:                       model.LiteLLMParams.MaxRetries,
				Organization:                     model.LiteLLMParams.Organization,
				RegionName:                       model.LiteLLMParams.RegionName,
				RPM:                              model.LiteLLMParams.RPM,
				StreamTimeout:                    model.LiteLLMParams.StreamTimeout,
				TPM:                              model.LiteLLMParams.TPM,
				Timeout:                          model.LiteLLMParams.Timeout,
				UseInPassThrough:                 model.LiteLLMParams.UseInPassThrough,
				UseLiteLLMProxy:                  model.LiteLLMParams.UseLiteLLMProxy,
				VertexProject:                    model.LiteLLMParams.VertexProject,
				VertexLocation:                   model.LiteLLMParams.VertexLocation,
				VertexCredentials:                model.LiteLLMParams.VertexCredentials,
				WatsonxRegionName:                model.LiteLLMParams.WatsonxRegionName,
			}

			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerToken, &litellmParams.OutputCostPerToken, "OutputCostPerToken"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerSecond, &litellmParams.OutputCostPerSecond, "OutputCostPerSecond"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.OutputCostPerPixel, &litellmParams.OutputCostPerPixel, "OutputCostPerPixel"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerPixel, &litellmParams.InputCostPerPixel, "InputCostPerPixel"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerSecond, &litellmParams.InputCostPerSecond, "InputCostPerSecond"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.InputCostPerToken, &litellmParams.InputCostPerSecond, "InputCostPerToken"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.MaxBudget, &litellmParams.MaxBudget, "MaxBudget"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}
			if err := parseAndAssign(model.LiteLLMParams.MaxFileSizeMB, &litellmParams.MaxFileSizeMB, "MaxFileSizeMB"); err != nil {
				log.Error(err, "parsing error")
				return "", err
			}

			if model.RequiresAuth {
				//get existing modelCredentials secret
				modelCredentials := &corev1.Secret{}
				err := k8sClient.Get(ctx, client.ObjectKey{Name: model.ModelCredentials.NameRef, Namespace: llm.Namespace}, modelCredentials)
				if err != nil {
					log.Error(err, "Model Credentials not found", "modelIdentifier", model.Identifier)
					return "", nil
				}

				//create individual secrets from the ModelCredentials
				secretNames, err := createAdditionalModelSecrets(ctx, &model, modelCredentials, llm, k8sClient, scheme)
				if err != nil {
					log.Error(err, "Failed to create secrets", "modelIdentifier", model.Identifier)
				}

				secretPrefix := "os.environ/"
				keys := model.ModelCredentials.Keys
				//match the secret to its yaml config to populate the yaml template
				fieldMap := map[string]*string{
					keys.VertexCredentials:  &litellmParams.VertexCredentials,
					keys.ApiBase:            &litellmParams.ApiBase,
					keys.AwsAccessKeyID:     &litellmParams.AwsAccessKeyID,
					keys.AwsSecretAccessKey: &litellmParams.AwsSecretAccessKey,
					keys.VertexProject:      &litellmParams.VertexCredentials,
					keys.ApiKey:             &litellmParams.ApiKey,
				}

				for key, target := range fieldMap {
					if key != "" {
						if secretName, ok := secretNames[key]; ok {
							*target = secretPrefix + secretName
						}
					}
				}

			}

			modelYAML := ModelListItemYAML{
				ModelName:     model.ModelName,
				LitellmParams: litellmParams,
			}
			modelListYAML = append(modelListYAML, modelYAML)
		}
	}

	var routerSettings RouterSettingsYAML
	if llm.Spec.RedisSecretRef.NameRef != "" {
		routerSettings = RouterSettingsYAML{
			Host:     llm.Spec.RedisSecretRef.Keys.HostSecret,
			Port:     llm.Spec.RedisSecretRef.Keys.PortSecret,
			Password: llm.Spec.RedisSecretRef.Keys.PasswordSecret,
		}
	}

	cfg := ProxyConfig{ModelList: modelListYAML, RouterSettings: routerSettings}
	cfg.GeneralSettings.AllowRequestsOnDBUnavailable = true
	cfg.GeneralSettings.StoreModelInDB = true

	b, _ := yaml.Marshal(cfg)
	return string(b), nil
}

// buildSecretData builds the secret data map for the LiteLLM instance.
// It handles the master key, either using the provided key or preserving an existing one.
func buildSecretData(masterKey string, existingSecret *corev1.Secret) map[string][]byte {
	data := make(map[string][]byte)

	// If no master key is provided, try to preserve existing one
	if masterKey == "" {
		if existingSecret != nil && existingSecret.Data != nil {
			if existingMasterKey, exists := existingSecret.Data["masterkey"]; exists {
				// Preserve existing master key
				data["masterkey"] = existingMasterKey
				return data
			}
		}
		// Generate a new random master key if none exists
		masterKey = uuid.New().String()
	}

	// Add master key (either provided or generated)
	if masterKey != "" {
		data["masterkey"] = []byte(masterKey)
	}

	return data
}

// updateStatus updates the LiteLLM instance status with current information.
// It updates observed generation, last updated timestamp, resource creation status, and conditions.
// Returns a combined status indicating whether all resources are ready.
func (r *LiteLLMInstanceReconciler) updateStatus(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance, deployment *appsv1.Deployment, service *corev1.Service) (bool, error) {
	// Update observed generation
	llm.Status.ObservedGeneration = llm.Generation

	// Update last updated timestamp
	now := metav1.Now()
	llm.Status.LastUpdated = &now

	// Update resource creation status
	llm.Status.ConfigMapCreated = true
	llm.Status.SecretCreated = true
	llm.Status.DeploymentCreated = deployment != nil
	llm.Status.ServiceCreated = service != nil
	llm.Status.IngressCreated = llm.Spec.Ingress.Enabled

	// Fetch the latest deployment status to ensure we have current information
	var latestDeployment *appsv1.Deployment
	if deployment != nil {
		latestDeployment = &appsv1.Deployment{}
		if err := r.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, latestDeployment); err != nil {
			// If we can't fetch the latest deployment, use the one we have
			latestDeployment = deployment
		}
	}

	// Update conditions with the latest deployment status
	r.updateConditions(llm, latestDeployment, service)

	if err := r.Status().Update(ctx, llm); err != nil {
		return false, err
	}

	// Determine if all resources are ready
	allReady := true

	// Check deployment readiness
	if latestDeployment != nil {
		expectedReplicas := int32(1)
		if latestDeployment.Spec.Replicas != nil {
			expectedReplicas = *latestDeployment.Spec.Replicas
		}

		if latestDeployment.Status.ReadyReplicas < expectedReplicas {
			allReady = false
		}
	} else {
		allReady = false
	}

	// Check service readiness
	if service == nil || service.Spec.ClusterIP == "" {
		allReady = false
	}

	// Check ingress readiness (if enabled)
	if llm.Spec.Ingress.Enabled && !llm.Status.IngressCreated {
		allReady = false
	}

	return allReady, nil
}

// updateConditions updates the conditions for the LiteLLM instance.
// It creates conditions for each resource type and an overall Ready condition.
func (r *LiteLLMInstanceReconciler) updateConditions(llm *litellmv1alpha1.LiteLLMInstance, deployment *appsv1.Deployment, service *corev1.Service) {
	now := metav1.Now()

	// ConfigMap ready condition
	configMapReady := metav1.Condition{
		Type:               "ConfigMapReady",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: llm.Generation,
		LastTransitionTime: now,
		Reason:             "ConfigMapNotReady",
		Message:            "ConfigMap is not ready",
	}

	if llm.Status.ConfigMapCreated {
		configMapReady.Status = metav1.ConditionTrue
		configMapReady.Reason = "ConfigMapReady"
		configMapReady.Message = "ConfigMap has been created"
	}

	// Secret ready condition
	secretReady := metav1.Condition{
		Type:               "SecretReady",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: llm.Generation,
		LastTransitionTime: now,
		Reason:             "SecretNotReady",
		Message:            "Secret is not ready",
	}

	if llm.Status.SecretCreated {
		secretReady.Status = metav1.ConditionTrue
		secretReady.Reason = "SecretReady"
		secretReady.Message = "Secret has been created"
	}

	// Deployment ready condition
	deploymentReady := metav1.Condition{
		Type:               "DeploymentReady",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: llm.Generation,
		LastTransitionTime: now,
		Reason:             "DeploymentNotReady",
		Message:            "Deployment is not ready",
	}

	// Check if deployment is actually ready
	if deployment != nil {
		expectedReplicas := int32(1) // Default to 1 replica as per createDeployment
		if deployment.Spec.Replicas != nil {
			expectedReplicas = *deployment.Spec.Replicas
		}

		if deployment.Status.ReadyReplicas >= expectedReplicas &&
			deployment.Status.AvailableReplicas >= expectedReplicas &&
			deployment.Status.UpdatedReplicas >= expectedReplicas {
			deploymentReady.Status = metav1.ConditionTrue
			deploymentReady.Reason = "DeploymentReady"
			deploymentReady.Message = fmt.Sprintf("Deployment has %d/%d ready replicas", deployment.Status.ReadyReplicas, expectedReplicas)
		} else {
			deploymentReady.Status = metav1.ConditionFalse
			deploymentReady.Reason = "DeploymentNotReady"
			deploymentReady.Message = fmt.Sprintf("Deployment has %d/%d ready replicas, %d/%d available, %d/%d updated",
				deployment.Status.ReadyReplicas, expectedReplicas,
				deployment.Status.AvailableReplicas, expectedReplicas,
				deployment.Status.UpdatedReplicas, expectedReplicas)
		}
	} else {
		deploymentReady.Status = metav1.ConditionFalse
		deploymentReady.Reason = "DeploymentNotReady"
		deploymentReady.Message = "Deployment object is nil"
	}

	// Service ready condition
	serviceReady := metav1.Condition{
		Type:               "ServiceReady",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: llm.Generation,
		LastTransitionTime: now,
		Reason:             "ServiceNotReady",
		Message:            "Service is not ready",
	}

	if service != nil && service.Spec.ClusterIP != "" {
		serviceReady.Status = metav1.ConditionTrue
		serviceReady.Reason = "ServiceReady"
		serviceReady.Message = fmt.Sprintf("Service is ready with ClusterIP: %s", service.Spec.ClusterIP)
	}

	// Ingress ready condition (only if ingress is enabled)
	if llm.Spec.Ingress.Enabled {
		ingressReady := metav1.Condition{
			Type:               "IngressReady",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: llm.Generation,
			LastTransitionTime: now,
			Reason:             "IngressNotReady",
			Message:            "Ingress is not ready",
		}

		if llm.Status.IngressCreated {
			ingressReady.Status = metav1.ConditionTrue
			ingressReady.Reason = "IngressReady"
			ingressReady.Message = "Ingress has been created"
		}

		r.setCondition(llm, ingressReady)
	}

	// Overall Ready condition
	ready := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: llm.Generation,
		LastTransitionTime: now,
		Reason:             "NotReady",
		Message:            "Not all resources are ready",
	}

	// Only set to True if ALL required resources are ready
	allReady := configMapReady.Status == metav1.ConditionTrue &&
		secretReady.Status == metav1.ConditionTrue &&
		deploymentReady.Status == metav1.ConditionTrue &&
		serviceReady.Status == metav1.ConditionTrue

	// If ingress is enabled, it must also be ready
	if llm.Spec.Ingress.Enabled {
		allReady = allReady && llm.Status.IngressCreated
	}

	if allReady {
		ready.Status = metav1.ConditionTrue
		ready.Reason = "Ready"
		ready.Message = "All resources are ready"
	}

	// Update or add conditions
	r.setCondition(llm, configMapReady)
	r.setCondition(llm, secretReady)
	r.setCondition(llm, deploymentReady)
	r.setCondition(llm, serviceReady)
	r.setCondition(llm, ready)
}

// setCondition sets a condition on the LiteLLM instance.
// It either updates an existing condition or adds a new one to the conditions slice.
func (r *LiteLLMInstanceReconciler) setCondition(llm *litellmv1alpha1.LiteLLMInstance, condition metav1.Condition) {
	for i, existingCondition := range llm.Status.Conditions {
		if existingCondition.Type == condition.Type {
			llm.Status.Conditions[i] = condition
			return
		}
	}
	llm.Status.Conditions = append(llm.Status.Conditions, condition)
}

// createConfigMap creates or updates the ConfigMap for the LiteLLM instance.
// It generates the LiteLLM proxy configuration and creates a ConfigMap containing the configuration file.
func (r *LiteLLMInstanceReconciler) createConfigMap(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.ConfigMap, error) {
	log := logf.FromContext(ctx)

	configYAML, err := renderProxyConfig(llm, ctx, r.Client, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to render proxy config")
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetConfigMapName(),
			Namespace: llm.Namespace,
		},
		Data: map[string]string{"proxy_server_config.yaml": configYAML},
	}

	if restart, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, configMap, llm); err != nil {
		log.Error(err, "Failed to create or update config map", "configMap", configMap.Name)
		return nil, err
	} else if restart {
		//get the deployment
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, client.ObjectKey{Name: r.litellmResourceNaming.GetDeploymentName(), Namespace: llm.Namespace}, deployment)
		if err != nil {
			return nil, err
		}
		log.Info("Restarting deployment", "deployment", deployment.Name)
		if err := util.RestartDeployment(ctx, r.Client, deployment.Name, deployment.Namespace); err != nil {
			log.Error(err, "Failed to restart deployment", "deployment", deployment.Name)
			return nil, err
		}
	}

	return configMap, nil
}

// sanitizes a key to a string that conforms with Kubernetes secret naming conventions
func sanitizeKey(key string) string {
	// Convert to lowercase
	key = strings.ToLower(key)

	// Replace any character that is not a-z, 0-9, or '-' with hyphen
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	sanitized := reg.ReplaceAllString(key, "-")

	// Trim leading/trailing hyphens or dots
	sanitized = strings.Trim(sanitized, "-.")

	// Ensure it starts and ends with an alphanumeric character
	startEndReg := regexp.MustCompile(`^[^a-z0-9]+|[^a-z0-9]+$`)
	sanitized = startEndReg.ReplaceAllString(sanitized, "")

	// Truncate to 253 characters if necessary
	if len(sanitized) > 253 {
		sanitized = sanitized[:253]
	}

	return sanitized
}

func createAdditionalModelSecrets(ctx context.Context, model *litellmv1alpha1.InitModelInstance, modelCredentials *corev1.Secret, llm *litellmv1alpha1.LiteLLMInstance,
	k8sClient client.Client, scheme *runtime.Scheme) (map[string]string, error) {

	log := logf.FromContext(ctx)
	secretNames := map[string]string{}
	for key, value := range modelCredentials.Data {
		secretName := sanitizeKey(fmt.Sprintf("%s_%s", model.Identifier, key))
		data := map[string][]byte{key: value}

		_, err := createSecret(ctx, k8sClient, scheme, secretName, modelCredentials.Namespace, data, llm)
		if err != nil {
			log.Error(err, "failed to create secret from ModelCredentials", "secretName", secretName)
			return nil, err
		}

		secretNames[key] = secretName
	}

	return secretNames, nil
}

// createSecret creates or updates the Secret for the LiteLLM instance.
// It creates a Secret containing the master key and other sensitive data.
func (r *LiteLLMInstanceReconciler) createMasterKeySecret(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.Secret, error) {

	// Get existing secret to preserve data if it exists
	secretName := r.litellmResourceNaming.GetSecretName()
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: llm.Namespace}, existingSecret)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	data := buildSecretData(llm.Spec.MasterKey, existingSecret)

	return createSecret(ctx, r.Client, r.Scheme, secretName, llm.Namespace, data, llm)
}

func createSecret(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, name string, namespace string, data map[string][]byte, owner client.Object) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, k8sClient, scheme, secret, owner); err != nil {
		return nil, err
	}

	return secret, nil
}

// createDeployment creates or updates the Deployment for the LiteLLM instance.
// It creates a Deployment that runs the LiteLLM proxy container with appropriate configuration.
func (r *LiteLLMInstanceReconciler) createDeployment(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance, configMap *corev1.ConfigMap, secret *corev1.Secret, serviceAccount *corev1.ServiceAccount) (*appsv1.Deployment, error) {
	log := logf.FromContext(ctx)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetDeploymentName(),
			Namespace: llm.Namespace,
			Labels:    r.litellmResourceNaming.GetAppLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: util.Int32Ptr(llm.Spec.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: r.litellmResourceNaming.GetAppLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.litellmResourceNaming.GetAppLabels(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount.Name,
					Containers: []corev1.Container{
						buildContainerSpec(llm, secret.Name, ctx, r.Client),
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMap.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	log.Info("Creating or updating deployment", "deployment", deployment.Name)
	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, deployment, llm); err != nil {
		log.Error(err, "Cannot create deployment")
		return nil, err
	}

	return deployment, nil
}

// createService creates or updates the Service for the LiteLLM instance.
// It creates a Service that exposes the LiteLLM proxy on port 80.
func (r *LiteLLMInstanceReconciler) createService(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetServiceName(),
			Namespace: llm.Namespace,
			Labels:    r.litellmResourceNaming.GetAppLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: r.litellmResourceNaming.GetAppLabels(),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       ServicePort,
					TargetPort: intstr.FromInt(ContainerPort),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, service, llm); err != nil {
		return nil, err
	}

	return service, nil
}

// createServiceAccount creates or updates the ServiceAccount for the LiteLLM instance.
// It creates a ServiceAccount that the Deployment will use.
func (r *LiteLLMInstanceReconciler) createServiceAccount(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetServiceAccountName(),
			Namespace: llm.Namespace,
			Labels:    r.litellmResourceNaming.GetAppLabels(),
		},
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, serviceAccount, llm); err != nil {
		return nil, err
	}

	return serviceAccount, nil
}

// createRBAC creates or updates the Role and RoleBinding for the LiteLLM instance ServiceAccount.
// It creates a Role with minimal permissions and a RoleBinding to bind it to the ServiceAccount.
func (r *LiteLLMInstanceReconciler) createRBAC(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance, serviceAccount *corev1.ServiceAccount) error {
	// Create Role with minimal permissions for the LiteLLM instance
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetRoleName(),
			Namespace: llm.Namespace,
			Labels:    r.litellmResourceNaming.GetAppLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, role, llm); err != nil {
		return err
	}

	// Create RoleBinding to bind the Role to the ServiceAccount
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetRoleBindingName(),
			Namespace: llm.Namespace,
			Labels:    r.litellmResourceNaming.GetAppLabels(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: serviceAccount.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, roleBinding, llm); err != nil {
		return err
	}

	return nil
}

// createIngress creates or updates the Ingress for the LiteLLM instance.
// It creates an Ingress resource to expose the LiteLLM proxy externally when enabled.
func (r *LiteLLMInstanceReconciler) createIngress(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.litellmResourceNaming.GetIngressName(),
			Namespace: llm.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: llm.Spec.Ingress.Host,
				},
			},
		},
	}

	if _, err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, ingress, llm); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
// It registers the controller with the controller-runtime manager and configures the watch.
func (r *LiteLLMInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&litellmv1alpha1.LiteLLMInstance{}).
		Named("litellm-litellminstance").
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

// buildContainerSpec builds the container specification for the LiteLLM deployment.
// It creates a complete container spec with image, arguments, ports, environment variables, and health checks.
func buildContainerSpec(llm *litellmv1alpha1.LiteLLMInstance, secretName string, ctx context.Context, k8sClient client.Client) corev1.Container {
	return corev1.Container{
		Name:  "litellm",
		Image: llm.Spec.Image,
		Args:  []string{"--config", ConfigPath},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: ContainerPort,
			},
		},
		Env: buildEnvironmentVariables(llm, secretName, ctx, k8sClient),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/etc/litellm",
				ReadOnly:  true,
			},
		},
		LivenessProbe:  buildLivenessProbe(),
		ReadinessProbe: buildReadinessProbe(),
		StartupProbe:   buildStartupProbe(),
	}
}

// buildEnvironmentVariables builds the environment variables for the container.
// It creates environment variables for the LiteLLM master key and database connection details.
func buildEnvironmentVariables(llm *litellmv1alpha1.LiteLLMInstance, secretName string, ctx context.Context, k8sClient client.Client) []corev1.EnvVar {
	log := logf.FromContext(ctx)
	envVars := []corev1.EnvVar{
		{
			Name: "LITELLM_MASTER_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "masterkey",
				},
			},
		},
	}

	// Add database environment variables if database secret reference is provided
	if llm.Spec.DatabaseSecretRef.NameRef != "" {
		dbEnvVars := []corev1.EnvVar{
			{
				Name: "DATABASE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: llm.Spec.DatabaseSecretRef.NameRef,
						},
						Key: llm.Spec.DatabaseSecretRef.Keys.DbnameSecret,
					},
				},
			},
			{
				Name: "DATABASE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: llm.Spec.DatabaseSecretRef.NameRef,
						},
						Key: llm.Spec.DatabaseSecretRef.Keys.HostSecret,
					},
				},
			},
			{
				Name: "DATABASE_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: llm.Spec.DatabaseSecretRef.NameRef,
						},
						Key: llm.Spec.DatabaseSecretRef.Keys.PasswordSecret,
					},
				},
			},
			{
				Name: "DATABASE_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: llm.Spec.DatabaseSecretRef.NameRef,
						},
						Key: llm.Spec.DatabaseSecretRef.Keys.UsernameSecret,
					},
				},
			},
		}
		envVars = append(envVars, dbEnvVars...)
	}

	for _, model := range llm.Spec.Models {
		if !model.RequiresAuth {
			log.Info("model requires no auth, so skipping secret mounting!")
			continue
		}

		prefix := sanitizeKey(model.Identifier)
		modelSecrets, err := getAllSecretsByPrefix(ctx, k8sClient, llm.Namespace, prefix)
		if err != nil {
			log.Error(err, "Failed to retrieve secrets using model unique identifier")
			return nil
		}

		//add all secrets we created into the EnvVars using valueFrom
		for _, secret := range modelSecrets {
			envVars = append(envVars, corev1.EnvVar{
				Name: secret.Name, //will be used in proxy_server_config.yaml to reference secret
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.Name,
						},
						Key: func() string {
							for key := range secret.Data {
								return key // use the first key in the secret data
							}
							err := errors.New("failed to retrieve secret key")
							log.Error(err, "No keys found in secret", "secretName", secret.Name)
							return ""
						}(),
					},
				},
			})
		}
	}

	return envVars
}

func getAllSecretsByPrefix(ctx context.Context, k8sClient client.Client, namespace string, prefix string) ([]corev1.Secret, error) {
	var allSecrets corev1.SecretList
	err := k8sClient.List(ctx, &allSecrets, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	var secrets []corev1.Secret
	for _, secret := range allSecrets.Items {
		if strings.HasPrefix(secret.Name, prefix) {
			secrets = append(secrets, secret)
		}
	}

	return secrets, nil
}

func getAllSecretsByPrefix(ctx context.Context, k8sClient client.Client, namespace string, prefix string) ([]corev1.Secret, error) {
	var allSecrets corev1.SecretList
	err := k8sClient.List(ctx, &allSecrets, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	var secrets []corev1.Secret
	for _, secret := range allSecrets.Items {
		if strings.HasPrefix(secret.Name, prefix) {
			secrets = append(secrets, secret)
		}
	}

	return secrets, nil
}

// buildLivenessProbe builds the liveness probe configuration.
// It creates a liveness probe that checks if the LiteLLM proxy is responding to health check requests.
func buildLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: LivenessPath,
				Port: util.FromInt(ContainerPort),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       30,
		TimeoutSeconds:      10,
		FailureThreshold:    3,
	}
}

// buildReadinessProbe builds the readiness probe configuration.
// It creates a readiness probe that checks if the LiteLLM proxy is ready to accept traffic.
func buildReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: ReadinessPath,
				Port: util.FromInt(ContainerPort),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
	}
}

// buildStartupProbe builds the startup probe configuration.
// It creates a startup probe that checks if the LiteLLM proxy has finished starting up.
func buildStartupProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: ReadinessPath,
				Port: util.FromInt(ContainerPort),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    30,
	}
}
