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

package v1alpha1

import (
	"github.com/bbdsoftware/litellm-operator/internal/interfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ModelSpec defines the desired state of Model.
type ModelSpec struct {
	// ConnectionRef is the connection reference
	ConnectionRef ConnectionRef `json:"connectionRef,omitempty"`

	// ModelName is the name of the model
	ModelName string `json:"modelName,omitempty"`

	// LiteLLMParams contains the LiteLLM parameters
	LiteLLMParams LiteLLMParams `json:"litellmParams,omitempty"`

	// ModelInfo contains the model information
	ModelInfo ModelInfo `json:"modelInfo,omitempty"`

	// ModelSecretRef is the model secret reference
	ModelSecretRef SecretRef `json:"modelSecretRef"`
}

// LiteLLMParams defines the LiteLLM parameters for a model.
type LiteLLMParams struct {
	// InputCostPerToken is the cost per input token
	InputCostPerToken *string `json:"inputCostPerToken,omitempty"`

	// OutputCostPerToken is the cost per output token
	OutputCostPerToken *string `json:"outputCostPerToken,omitempty"`

	// InputCostPerSecond is the cost per second for input
	InputCostPerSecond *string `json:"inputCostPerSecond,omitempty"`

	// OutputCostPerSecond is the cost per second for output
	OutputCostPerSecond *string `json:"outputCostPerSecond,omitempty"`

	// InputCostPerPixel is the cost per pixel for input
	InputCostPerPixel *string `json:"inputCostPerPixel,omitempty"`

	// OutputCostPerPixel is the cost per pixel for output
	OutputCostPerPixel *string `json:"outputCostPerPixel,omitempty"`

	// APIKey is the API key for the model
	ApiKey *string `json:"apiKey,omitempty"`

	// APIBase is the base URL for the API
	ApiBase *string `json:"apiBase,omitempty"`

	// APIVersion is the version of the API
	ApiVersion *string `json:"apiVersion,omitempty"`

	// VertexProject is the Google Cloud project for Vertex AI
	VertexProject *string `json:"vertexProject,omitempty"`

	// VertexLocation is the location for Vertex AI
	VertexLocation *string `json:"vertexLocation,omitempty"`

	// VertexCredentials is the credentials for Vertex AI
	VertexCredentials *string `json:"vertexCredentials,omitempty"`

	// RegionName is the region name for the service
	RegionName *string `json:"regionName,omitempty"`

	// AWSAccessKeyID is the AWS access key ID
	AwsAccessKeyID *string `json:"awsAccessKeyId,omitempty"`

	// AWSSecretAccessKey is the AWS secret access key
	AwsSecretAccessKey *string `json:"awsSecretAccessKey,omitempty"`

	// AWSRegionName is the AWS region name
	AwsRegionName *string `json:"awsRegionName,omitempty"`

	// WatsonXRegionName is the WatsonX region name
	WatsonXRegionName *string `json:"watsonxRegionName,omitempty"`

	// CustomLLMProvider is the custom LLM provider
	CustomLLMProvider *string `json:"customLLMProvider,omitempty"`

	// TPM is tokens per minute
	TPM *int `json:"tpm,omitempty"`

	// RPM is requests per minute
	RPM *int `json:"rpm,omitempty"`

	// Timeout is the timeout in seconds
	Timeout *int `json:"timeout,omitempty"`

	// StreamTimeout is the stream timeout in seconds
	StreamTimeout *int `json:"streamTimeout,omitempty"`

	// MaxRetries is the maximum number of retries
	MaxRetries *int `json:"maxRetries,omitempty"`

	// Organization is the organization name
	Organization *string `json:"organization,omitempty"`

	// ConfigurableClientsideAuthParams are configurable client-side auth parameters
	ConfigurableClientsideAuthParams *[]runtime.RawExtension `json:"configurableClientsideAuthParams,omitempty"`

	// LiteLLMCredentialName is the LiteLLM credential name
	LiteLLMCredentialName *string `json:"litellmCredentialName,omitempty"`

	// LiteLLMTraceID is the LiteLLM trace ID
	LiteLLMTraceID *string `json:"litellmTraceId,omitempty"`

	// MaxFileSizeMB is the maximum file size in MB
	MaxFileSizeMB *int `json:"maxFileSizeMb,omitempty"`

	// MaxBudget is the maximum budget
	MaxBudget *string `json:"maxBudget,omitempty"`

	// BudgetDuration is the budget duration
	BudgetDuration *string `json:"budgetDuration,omitempty"`

	// UseInPassThrough indicates if to use in pass through
	UseInPassThrough *bool `json:"useInPassThrough,omitempty"`

	// UseLiteLLMProxy indicates if to use LiteLLM proxy
	UseLiteLLMProxy *bool `json:"useLitellmProxy,omitempty"`

	// MergeReasoningContentInChoices indicates if to merge reasoning content in choices
	MergeReasoningContentInChoices *bool `json:"mergeReasoningContentInChoices,omitempty"`

	// ModelInfo contains additional model information
	ModelInfo *runtime.RawExtension `json:"modelInfo,omitempty"`

	// MockResponse is the mock response
	MockResponse *string `json:"mockResponse,omitempty"`

	// AutoRouterConfigPath is the auto router config path
	AutoRouterConfigPath *string `json:"autoRouterConfigPath,omitempty"`

	// AutoRouterConfig is the auto router config
	AutoRouterConfig *string `json:"autoRouterConfig,omitempty"`

	// AutoRouterDefaultModel is the auto router default model
	AutoRouterDefaultModel *string `json:"autoRouterDefaultModel,omitempty"`

	// AutoRouterEmbeddingModel is the auto router embedding model
	AutoRouterEmbeddingModel *string `json:"autoRouterEmbeddingModel,omitempty"`

	// Model is the model name
	Model *string `json:"model,omitempty"`

	// AdditionalProps contains additional properties
	AdditionalProps *runtime.RawExtension `json:"additionalProp1,omitempty"`
}

// ModelInfo defines the model information.
type ModelInfo struct {
	// ID is the model ID
	ID *string `json:"id,omitempty"`

	// DBModel indicates if this is a database model
	DBModel *bool `json:"dbModel,omitempty"`

	// TeamID is the team ID
	TeamID *string `json:"teamId,omitempty"`

	// TeamPublicModelName is the team public model name
	TeamPublicModelName *string `json:"teamPublicModelName,omitempty"`

	// AdditionalProps contains additional properties
	AdditionalProps *runtime.RawExtension `json:"additionalProp1,omitempty"`
}

// ModelStatus defines the observed state of Model.
type ModelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// LastUpdated represents the last time the status was updated
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
	// Conditions represent the latest available observations of a LiteLLM instance's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ModelName is the name of the model
	ModelName *string `json:"modelName,omitempty"`

	// LiteLLMParams contains the LiteLLM parameters
	LiteLLMParams *LiteLLMParams `json:"litellmParams,omitempty"`

	// ModelId contains the model uuid provided by litellm server
	ModelId *string `json:"modelId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Model Name",type="string",JSONPath=".spec.modelName",description="Name of the model"
// +kubebuilder:printcolumn:name="Connection",type="string",JSONPath=".spec.connectionRef.secretRef.secretName",description="Connection secret name"
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.litellmParams.customLLMProvider",description="LLM provider"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the model"
// +kubebuilder:printcolumn:name="Model ID",type="string",JSONPath=".status.modelId",description="Model UUID from LiteLLM server"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Ready status"

// Model is the Schema for the models API.
type Model struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelSpec   `json:"spec,omitempty"`
	Status ModelStatus `json:"status,omitempty"`
}

type ConnectionRef struct {
	SecretRef   SecretRef   `json:"secretRef,omitempty"`
	InstanceRef InstanceRef `json:"instanceRef,omitempty"`
}

type SecretRef struct {
	Namespace  string `json:"namespace,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}

type InstanceRef struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// NilKeys implements KeysInterface for types that don't have keys
type NilKeys struct{}

// GetMasterKey returns empty string for nil keys
func (n NilKeys) GetMasterKey() string {
	return ""
}

// GetURL returns empty string for nil keys
func (n NilKeys) GetURL() string {
	return ""
}

// GetSecretRef returns the SecretRef if it exists
func (c ConnectionRef) GetSecretRef() interface{} {
	// Check if SecretRef is not empty (has either Namespace or SecretName)
	if c.SecretRef.Namespace != "" || c.SecretRef.SecretName != "" {
		return c.SecretRef
	}
	return nil
}

// GetInstanceRef returns the LitellmInstanceRef if it exists
func (c ConnectionRef) GetInstanceRef() interface{} {
	// Check if LitellmInstanceRef is not empty (has either Namespace or LitellmInstanceName)
	if c.InstanceRef.Namespace != "" || c.InstanceRef.Name != "" {
		return c.InstanceRef
	}
	return nil
}

// HasSecretRef checks if SecretRef is specified
func (c ConnectionRef) HasSecretRef() bool {
	return c.SecretRef.Namespace != "" || c.SecretRef.SecretName != ""
}

// HasInstanceRef checks if LitellmInstanceRef is specified
func (c ConnectionRef) HasInstanceRef() bool {
	return c.InstanceRef.Namespace != "" || c.InstanceRef.Name != ""
}

// GetSecretName returns the secret name
func (s SecretRef) GetSecretName() string {
	return s.SecretName
}

// GetNamespace returns the namespace
func (s SecretRef) GetNamespace() string {
	return s.Namespace
}

// GetKeys returns NilKeys since litellm SecretRef doesn't have keys structure
func (s SecretRef) GetKeys() interfaces.KeysInterface {
	return NilKeys{}
}

// HasKeys returns false since litellm SecretRef doesn't have keys structure
func (s SecretRef) HasKeys() bool {
	return false
}

// GetInstanceName returns the instance name
func (i InstanceRef) GetInstanceName() string {
	return i.Name
}

// GetNamespace returns the namespace
func (i InstanceRef) GetNamespace() string {
	return i.Namespace
}

// +kubebuilder:object:root=true

// ModelList contains a list of Model.
type ModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Model `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Model{}, &ModelList{})
}
