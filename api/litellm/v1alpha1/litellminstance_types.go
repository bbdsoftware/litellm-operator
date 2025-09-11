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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// LiteLLMInstanceSpec defines the desired state of LiteLLMInstance.
type LiteLLMInstanceSpec struct {
	// +kubebuilder:default="ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1"
	Image             string            `json:"image"`
	MasterKey         string            `json:"masterKey,omitempty"`
	DatabaseSecretRef DatabaseSecretRef `json:"databaseSecretRef,omitempty"`
	RedisSecretRef    RedisSecretRef    `json:"redisSecretRef,omitempty"`
	Ingress           Ingress           `json:"ingress,omitempty"`
	Gateway           Gateway           `json:"gateway,omitempty"`

	// +kubebuilder:default=1
	Replicas     int32               `json:"replicas,omitempty"`
	Models       []InitModelInstance `json:"models,omitempty"`
	ExtraEnvVars []corev1.EnvVar     `json:"extraEnvVars,omitempty"`
}

// model instance used to create proxy server config map
type InitModelInstance struct {
	ModelName            string                   `json:"modelName,omitempty"`
	RequiresAuth         bool                     `json:"requiresAuth"`
	Identifier           string                   `json:"identifier"`
	ModelCredentials     ModelCredentialSecretRef `json:"modelCredentials,omitempty"`
	LiteLLMParams        LiteLLMParams            `json:"liteLLMParams,omitempty"`
	AdditionalProperties runtime.RawExtension     `json:"additionalProperties,omitempty"`
}

type ModelCredentialSecretRef struct {
	NameRef string                    `json:"nameRef"`
	Keys    ModelCredentialSecretKeys `json:"keys"`
}

type ModelCredentialSecretKeys struct {
	ApiKey             string `json:"apiKey,omitempty"`
	ApiBase            string `json:"apiBase,omitempty"`
	AwsSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`
	AwsAccessKeyID     string `json:"awsAccessKeyId,omitempty"`
	VertexCredentials  string `json:"vertexCredentials,omitempty"`
	VertexProject      string `json:"vertexProject,omitempty"`
}

type DatabaseSecretRef struct {
	NameRef string             `json:"nameRef"`
	Keys    DatabaseSecretKeys `json:"keys"`
}

type RedisSecretRef struct {
	NameRef string          `json:"nameRef"`
	Keys    RedisSecretKeys `json:"keys"`
}

type Ingress struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
}

type Gateway struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
}
type DatabaseSecretKeys struct {
	HostSecret     string `json:"hostSecret"`
	PasswordSecret string `json:"passwordSecret"`
	UsernameSecret string `json:"usernameSecret"`
	DbnameSecret   string `json:"dbnameSecret"`
}

type RedisSecretKeys struct {
	HostSecret     string `json:"hostSecret"`
	PortSecret     string `json:"portSecret"`
	PasswordSecret string `json:"passwordSecret"`
}

// LiteLLMInstanceStatus defines the observed state of LiteLLMInstance.
type LiteLLMInstanceStatus struct {
	// ObservedGeneration represents the .metadata.generation that the condition was set based upon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastUpdated represents the last time the status was updated
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Resource creation status
	ConfigMapCreated  bool `json:"configMapCreated,omitempty"`
	SecretCreated     bool `json:"secretCreated,omitempty"`
	DeploymentCreated bool `json:"deploymentCreated,omitempty"`
	ServiceCreated    bool `json:"serviceCreated,omitempty"`
	IngressCreated    bool `json:"ingressCreated,omitempty"`

	// Conditions represent the latest available observations of a LiteLLM instance's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="The ready status of the instance"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="Number of replicas"
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image",description="The LiteLLM image being used"
// +kubebuilder:printcolumn:name="Database",type="string",JSONPath=".spec.databaseSecretRef.nameRef",description="Database secret reference"
// +kubebuilder:printcolumn:name="Ingress",type="boolean",JSONPath=".spec.ingress.enabled",description="Whether ingress is enabled"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time since creation"

// LiteLLMInstance is the Schema for the litellminstances API.
type LiteLLMInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LiteLLMInstanceSpec   `json:"spec,omitempty"`
	Status LiteLLMInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LiteLLMInstanceList contains a list of LiteLLMInstance.
type LiteLLMInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LiteLLMInstance `json:"items"`
}

// GetConditions returns the conditions slice
func (m *LiteLLMInstance) GetConditions() []metav1.Condition {
	return m.Status.Conditions
}

// SetConditions sets the conditions slice
func (m *LiteLLMInstance) SetConditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&LiteLLMInstance{}, &LiteLLMInstanceList{})
}
