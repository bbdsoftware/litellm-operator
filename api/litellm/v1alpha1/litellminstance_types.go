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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	PortSecret     int    `json:"portSecret"`
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
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image",description="The LiteLLM image being used"
// +kubebuilder:printcolumn:name="Redis",type="string",JSONPath=".spec.redisSecretRef.nameRef",description="Redis secret reference"
// +kubebuilder:printcolumn:name="Ingress",type="string",JSONPath=".spec.ingress.enabled",description="Whether ingress is enabled"
// +kubebuilder:printcolumn:name="Gateway",type="string",JSONPath=".spec.gateway.enabled",description="Whether gateway is enabled"
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".status.secretCreated",description="Secret creation status"
// +kubebuilder:printcolumn:name="Deployment",type="string",JSONPath=".status.deploymentCreated",description="Deployment creation status"
// +kubebuilder:printcolumn:name="Service",type="string",JSONPath=".status.serviceCreated",description="Service creation status"
// +kubebuilder:printcolumn:name="Ingress Created",type="string",JSONPath=".status.ingressCreated",description="Ingress creation status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the LiteLLM instance"

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

func init() {
	SchemeBuilder.Register(&LiteLLMInstance{}, &LiteLLMInstanceList{})
}
