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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/util"
	"github.com/google/uuid"
	rbacv1 "k8s.io/api/rbac/v1"
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
	Scheme *runtime.Scheme
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

	// Create or update resources
	configMap, err := r.createConfigMap(ctx, llm)
	if err != nil {
		log.Error(err, "Failed to create or update ConfigMap")
		return util.HandleConflictError(err)
	}

	secret, err := r.createSecret(ctx, llm)
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

// renderProxyConfig generates the YAML configuration for the LiteLLM proxy server.
// It creates a configuration structure with model list, router settings, and general settings.
func renderProxyConfig(llm *litellmv1alpha1.LiteLLMInstance) string {
	// Create a custom struct for YAML output that handles API key references
	type ModelEntryYAML struct {
		ModelName     string `yaml:"model_name"`
		LitellmParams struct {
			Model   string `yaml:"model"`
			APIBase string `yaml:"api_base"`
			APIKey  string `yaml:"api_key"`
			RPM     *int   `yaml:"rpm,omitempty"`
		} `yaml:"litellm_params"`
	}

	type RouterSettingsYAML struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Password string `yaml:"password,omitempty"`
	}

	type ProxyConfig struct {
		ModelList       []ModelEntryYAML   `yaml:"model_list"`
		RouterSettings  RouterSettingsYAML `yaml:"router_settings,omitempty"`
		GeneralSettings struct {
			AllowRequestsOnDBUnavailable bool `yaml:"allow_requests_on_db_unavailable"`
		} `yaml:"general_settings"`
	}

	var modelListYAML []ModelEntryYAML
	// if llm.Spec.ModelList != nil {
	// 	for _, model := range llm.Spec.ModelList {
	// 		modelYAML := ModelEntryYAML{
	// 			ModelName: model.ModelName,
	// 		}
	// 		modelYAML.LitellmParams.Model = model.LitellmParams.Model
	// 		modelYAML.LitellmParams.APIBase = model.LitellmParams.APIBase
	// 		modelYAML.LitellmParams.RPM = model.LitellmParams.RPM

	// 		// Reference API key from environment variable
	// 		if model.LitellmParams.APIKey != "" {
	// 			modelYAML.LitellmParams.APIKey = fmt.Sprintf("os.environ/%s_API_KEY", model.ModelName)
	// 		}

	// 		modelListYAML = append(modelListYAML, modelYAML)
	// 	}
	// }

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
	b, _ := yaml.Marshal(cfg)
	return string(b)
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
	configYAML := renderProxyConfig(llm)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetConfigMapName(llm.Name),
			Namespace: llm.Namespace,
		},
		Data: map[string]string{"proxy_server_config.yaml": configYAML},
	}

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, configMap, llm); err != nil {
		return nil, err
	}

	return configMap, nil
}

// createSecret creates or updates the Secret for the LiteLLM instance.
// It creates a Secret containing the master key and other sensitive data.
func (r *LiteLLMInstanceReconciler) createSecret(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetSecretName(llm.Name),
			Namespace: llm.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
	}

	// Get existing secret to preserve data if it exists
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, existingSecret)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	secret.Data = buildSecretData(llm.Spec.MasterKey, existingSecret)

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, secret, llm); err != nil {
		return nil, err
	}

	return secret, nil
}

// createDeployment creates or updates the Deployment for the LiteLLM instance.
// It creates a Deployment that runs the LiteLLM proxy container with appropriate configuration.
func (r *LiteLLMInstanceReconciler) createDeployment(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance, configMap *corev1.ConfigMap, secret *corev1.Secret, serviceAccount *corev1.ServiceAccount) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetDeploymentName(llm.Name),
			Namespace: llm.Namespace,
			Labels:    util.GetAppLabels(llm.Name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: util.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: util.GetAppLabels(llm.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: util.GetAppLabels(llm.Name),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount.Name,
					Containers: []corev1.Container{
						buildContainerSpec(llm, secret.Name),
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

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, deployment, llm); err != nil {
		return nil, err
	}

	return deployment, nil
}

// createService creates or updates the Service for the LiteLLM instance.
// It creates a Service that exposes the LiteLLM proxy on port 80.
func (r *LiteLLMInstanceReconciler) createService(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetServiceName(llm.Name),
			Namespace: llm.Namespace,
			Labels:    util.GetAppLabels(llm.Name),
		},
		Spec: corev1.ServiceSpec{
			Selector: util.GetAppLabels(llm.Name),
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

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, service, llm); err != nil {
		return nil, err
	}

	return service, nil
}

// createServiceAccount creates or updates the ServiceAccount for the LiteLLM instance.
// It creates a ServiceAccount that the Deployment will use.
func (r *LiteLLMInstanceReconciler) createServiceAccount(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetServiceAccountName(llm.Name),
			Namespace: llm.Namespace,
			Labels:    util.GetAppLabels(llm.Name),
		},
	}

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, serviceAccount, llm); err != nil {
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
			Name:      util.GetRoleName(llm.Name),
			Namespace: llm.Namespace,
			Labels:    util.GetAppLabels(llm.Name),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if err := util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, role, llm); err != nil {
		return err
	}

	// Create RoleBinding to bind the Role to the ServiceAccount
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetRoleBindingName(llm.Name),
			Namespace: llm.Namespace,
			Labels:    util.GetAppLabels(llm.Name),
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

	return util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, roleBinding, llm)
}

// createIngress creates or updates the Ingress for the LiteLLM instance.
// It creates an Ingress resource to expose the LiteLLM proxy externally when enabled.
func (r *LiteLLMInstanceReconciler) createIngress(ctx context.Context, llm *litellmv1alpha1.LiteLLMInstance) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetIngressName(llm.Name),
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

	return util.CreateOrUpdateWithRetry(ctx, r.Client, r.Scheme, ingress, llm)
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
func buildContainerSpec(llm *litellmv1alpha1.LiteLLMInstance, secretName string) corev1.Container {
	return corev1.Container{
		Name:  "litellm",
		Image: llm.Spec.Image,
		Args:  []string{"--config", ConfigPath},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: ContainerPort,
			},
		},
		Env: buildEnvironmentVariables(llm, secretName),
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
func buildEnvironmentVariables(llm *litellmv1alpha1.LiteLLMInstance, secretName string) []corev1.EnvVar {
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

	return envVars
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
