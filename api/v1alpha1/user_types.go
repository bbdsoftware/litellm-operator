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

// UserSpec defines the desired state of User
type UserSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// UserID is the ID of the user. If not set, a unique ID will be generated.
	UserID string `json:"userID,omitempty"`
	// UserAlias is the alias of the user
	UserAlias string `json:"userAlias,omitempty"`
	// UserEmail is the email of the user
	UserEmail string `json:"userEmail,omitempty"`
	// UserRole is the role of the user - one of "proxy_admin", "proxy_admin_viewer", "internal_user", "internal_user_viewer", "team", "customer"
	UserRole string `json:"userRole,omitempty"`
	// SendInviteEmail is whether to send an invite email to the user - NOTE: the user endpoint will return an error if email alerting is not configured and this is enabled, but the user will still be created.
	SendInviteEmail bool `json:"sendInviteEmail,omitempty"`
	// Teams is the list of teams that the user is a member of
	Teams []string `json:"teams,omitempty"`
	// AutoCreateKey is whether to automatically create a key for the user
	AutoCreateKey bool `json:"autoCreateKey,omitempty"`
	// KeyAlias is the optional alias of the key if autoCreateKey is true
	KeyAlias string `json:"keyAlias,omitempty"`
	// SoftBudget - alert when user exceeds this budget, doesn't block requests
	SoftBudget string `json:"softBudget,omitempty"`
	// MaxBudget is the maximum budget for the user
	MaxBudget string `json:"maxBudget,omitempty"`
	// ModelMaxBudget is the model specific maximum budget
	ModelMaxBudget map[string]string `json:"modelMaxBudget,omitempty"`
	// ModelRPMLimit is the model specific maximum requests per minute
	ModelRPMLimit map[string]string `json:"modelRPMLimit,omitempty"`
	// ModelTPMLimit is the model specific maximum tokens per minute
	ModelTPMLimit map[string]string `json:"modelTPMLimit,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo").
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// Models is the list of models that the user is allowed to use
	Models []string `json:"models,omitempty"`
	// MaxParallelRequests is the maximum number of parallel requests for the user
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// Metadata is the metadata of the user
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CreatedAt is the date and time when the key was created
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the date and time when the key was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// Expires is the date and time when the user will expire
	Expires string `json:"expires,omitempty"`
	// UserID is the unique user id - used for tracking spend across multiple keys for same user id.
	UserID string `json:"userID,omitempty"`
	// UserAlias is the alias of the user
	UserAlias string `json:"userAlias,omitempty"`
	// UserEmail is the email of the user
	UserEmail string `json:"userEmail,omitempty"`
	// UserRole is the role of the user - one of "proxy_admin", "proxy_admin_viewer", "internal_user", "internal_user_viewer", "team", "customer"
	UserRole string `json:"userRole,omitempty"`
	// Teams is the list of teams that the user is a member of
	Teams []string `json:"teams,omitempty"`
	// KeyAlias is the optional alias of the key if autoCreateKey is true
	KeyAlias string `json:"keyAlias,omitempty"`
	// MaxBudget is the maximum budget for the user
	MaxBudget string `json:"maxBudget,omitempty"`
	// ModelMaxBudget is the model specific maximum budget
	ModelMaxBudget map[string]string `json:"modelMaxBudget,omitempty"`
	// ModelRPMLimit is the model specific maximum requests per minute
	ModelRPMLimit map[string]string `json:"modelRPMLimit,omitempty"`
	// ModelTPMLimit is the model specific maximum tokens per minute
	ModelTPMLimit map[string]string `json:"modelTPMLimit,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo").
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// Models is the list of models that the user is allowed to use
	Models []string `json:"models,omitempty"`
	// MaxParallelRequests is the maximum number of parallel requests for the user
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// Metadata is the metadata of the user
	Metadata map[string]string `json:"metadata,omitempty"`
	// SecretRef is the reference to the secret containing the user key
	SecretRef string `json:"secretRef,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
