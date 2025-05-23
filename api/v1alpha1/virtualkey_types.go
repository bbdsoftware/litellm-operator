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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualKeySpec defines the desired state of VirtualKey
type VirtualKeySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// KeyAlias is the user defined key alias
	KeyAlias string `json:"keyAlias,omitempty"`
	// Model names that a user is allowed to call. If empty, all models are allowed.
	Models []string `json:"models,omitempty"`
	// TeamID is the team ID for the key
	TeamID string `json:"teamID,omitempty"`
	// MaxBudget is the maximum budget for the key
	MaxBudget string `json:"maxBudget,omitempty"`
	// BudgetDuration is the duration of the budget
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// UserID is the user ID of the key
	UserID string `json:"userID,omitempty"`
	// Metadata is the metadata of the key
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VirtualKeyStatus defines the observed state of VirtualKey
type VirtualKeyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CreatedAt is the date and time when the key was created
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the date and time when the key was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// KeyAlias is the user defined key alias
	KeyAlias string `json:"keyAlias,omitempty"`
	// KeyID is the generated ID of the key
	KeyID string `json:"keyID,omitempty"`
	// KeyName is the redacted secret key
	KeyName string `json:"keyName,omitempty"`
	// Expires is the date and time when the key will expire
	Expires string `json:"expires,omitempty"`
	// SecretRef is the reference to the secret containing the key
	SecretRef string `json:"secretRef,omitempty"`
	// MaxBudget is the maximum budget for the key
	MaxBudget string `json:"maxBudget,omitempty"`
	// BudgetDuration is the duration of the budget
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// Model names that a user is allowed to call. If empty, all models are allowed.
	Models []string `json:"models,omitempty"`
	// UserID is the unique user id - used for tracking spend across multiple keys for same user id.
	UserID string `json:"userID,omitempty"`
	// Metadata is the metadata of the key
	Metadata map[string]string `json:"metadata,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualKey is the Schema for the virtualkeys API
type VirtualKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualKeySpec   `json:"spec,omitempty"`
	Status VirtualKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualKeyList contains a list of VirtualKey
type VirtualKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualKey{}, &VirtualKeyList{})
}
