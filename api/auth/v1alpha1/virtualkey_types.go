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
	// ConnectionRef defines how to connect to the LiteLLM instance
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// Aliases maps additional aliases for the key
	Aliases map[string]string `json:"aliases,omitempty"`
	// AllowedCacheControls defines allowed cache control settings
	AllowedCacheControls []string `json:"allowedCacheControls,omitempty"`
	// AllowedRoutes defines allowed API routes
	AllowedRoutes []string `json:"allowedRoutes,omitempty"`
	// Blocked indicates if the key is blocked
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration specifies the duration for budget tracking
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// BudgetID is the identifier for the budget
	BudgetID string `json:"budgetID,omitempty"`
	// Config contains additional configuration settings
	Config map[string]string `json:"config,omitempty"`
	// Duration specifies how long the key is valid
	Duration string `json:"duration,omitempty"`
	// EnforcedParams lists parameters that must be included in requests
	EnforcedParams []string `json:"enforcedParams,omitempty"`
	// Guardrails defines guardrail settings
	Guardrails []string `json:"guardrails,omitempty"`
	// Key is the actual key value
	Key string `json:"key,omitempty"`
	// KeyAlias is the user defined key alias
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="KeyAlias is immutable"
	KeyAlias string `json:"keyAlias,omitempty"`
	// MaxBudget sets the maximum budget limit
	MaxBudget string `json:"maxBudget,omitempty"`
	// MaxParallelRequests limits concurrent requests
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// Metadata contains additional metadata
	Metadata map[string]string `json:"metadata,omitempty"`
	// ModelMaxBudget sets budget limits per model
	ModelMaxBudget map[string]string `json:"modelMaxBudget,omitempty"`
	// ModelRPMLimit sets RPM limits per model
	ModelRPMLimit map[string]int `json:"modelRPMLimit,omitempty"`
	// ModelTPMLimit sets TPM limits per model
	ModelTPMLimit map[string]int `json:"modelTPMLimit,omitempty"`
	// Models specifies which models can be used
	Models []string `json:"models,omitempty"`
	// Permissions defines key permissions
	Permissions map[string]string `json:"permissions,omitempty"`
	// RPMLimit sets global RPM limit
	RPMLimit int `json:"rpmLimit,omitempty"`
	// SendInviteEmail indicates whether to send an invite email
	SendInviteEmail bool `json:"sendInviteEmail,omitempty"`
	// SoftBudget sets a soft budget limit
	SoftBudget string `json:"softBudget,omitempty"`
	// Spend tracks the current spend amount
	Spend string `json:"spend,omitempty"`
	// Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license
	Tags []string `json:"tags,omitempty"`
	// TeamID identifies the team associated with the key
	TeamID string `json:"teamID,omitempty"`
	// TPMLimit sets global TPM limit
	TPMLimit int `json:"tpmLimit,omitempty"`
	// UserID identifies the user associated with the key
	UserID string `json:"userID,omitempty"`
}

// VirtualKeyStatus defines the observed state of VirtualKey
type VirtualKeyStatus struct {
	// Aliases maps additional aliases for the key
	Aliases map[string]string `json:"aliases,omitempty"`
	// AllowedCacheControls defines allowed cache control settings
	AllowedCacheControls []string `json:"allowedCacheControls,omitempty"`
	// AllowedRoutes defines allowed API routes
	AllowedRoutes []string `json:"allowedRoutes,omitempty"`
	// Blocked indicates if the key is blocked
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration is the duration of the budget
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// BudgetID is the identifier for the budget
	BudgetID string `json:"budgetID,omitempty"`
	// BudgetResetAt is the date and time when the budget will reset
	BudgetResetAt string `json:"budgetResetAt,omitempty"`
	// Config contains additional configuration settings
	Config map[string]string `json:"config,omitempty"`
	// CreatedAt is the date and time when the key was created
	CreatedAt string `json:"createdAt,omitempty"`
	// CreatedBy tracks who created the key
	CreatedBy string `json:"createdBy,omitempty"`
	// Duration specifies how long the key is valid
	Duration string `json:"duration,omitempty"`
	// EnforcedParams lists parameters that must be included in requests
	EnforcedParams []string `json:"enforcedParams,omitempty"`
	// Expires is the date and time when the key will expire
	Expires string `json:"expires,omitempty"`
	// Guardrails defines guardrail settings
	Guardrails []string `json:"guardrails,omitempty"`
	// KeyAlias is the user defined key alias
	KeyAlias string `json:"keyAlias,omitempty"`
	// KeyID is the generated ID of the key
	KeyID string `json:"keyID,omitempty"`
	// KeyName is the redacted secret key
	KeyName string `json:"keyName,omitempty"`
	// KeySecretRef is the reference to the secret containing the key
	KeySecretRef string `json:"keySecretRef,omitempty"`
	// LiteLLMBudgetTable is the budget table reference
	LiteLLMBudgetTable string `json:"liteLLMBudgetTable,omitempty"`
	// MaxBudget is the maximum budget for the key
	MaxBudget string `json:"maxBudget,omitempty"`
	// MaxParallelRequests limits concurrent requests
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// Models specifies which models can be used
	Models []string `json:"models,omitempty"`
	// Permissions defines key permissions
	Permissions map[string]string `json:"permissions,omitempty"`
	// RPMLimit sets global RPM limit
	RPMLimit int `json:"rpmLimit,omitempty"`
	// Spend tracks the current spend amount
	Spend string `json:"spend,omitempty"`
	// Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license
	Tags []string `json:"tags,omitempty"`
	// TeamID identifies the team associated with the key
	TeamID string `json:"teamID,omitempty"`
	// Token contains the actual API key
	Token string `json:"token,omitempty"`
	// TokenID is the unique identifier for the token
	TokenID string `json:"tokenID,omitempty"`
	// TPMLimit sets global TPM limit
	TPMLimit int `json:"tpmLimit,omitempty"`
	// UpdatedAt is the date and time when the key was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// UpdatedBy tracks who last updated the key
	UpdatedBy string `json:"updatedBy,omitempty"`
	// UserID identifies the user associated with the key
	UserID string `json:"userID,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Key ID",type="string",JSONPath=".status.keyID",description="The unique key identifier"
// +kubebuilder:printcolumn:name="Key Alias",type="string",JSONPath=".spec.keyAlias",description="The key alias"
// +kubebuilder:printcolumn:name="User ID",type="string",JSONPath=".spec.userID",description="The associated user ID"
// +kubebuilder:printcolumn:name="Team ID",type="string",JSONPath=".spec.teamID",description="The associated team ID"
// +kubebuilder:printcolumn:name="Blocked",type="boolean",JSONPath=".spec.blocked",description="Whether the key is blocked"
// +kubebuilder:printcolumn:name="Max Budget",type="string",JSONPath=".spec.maxBudget",description="Maximum budget for the key"
// +kubebuilder:printcolumn:name="RPM Limit",type="integer",JSONPath=".spec.rpmLimit",description="Requests per minute limit"
// +kubebuilder:printcolumn:name="TPM Limit",type="integer",JSONPath=".spec.tpmLimit",description="Tokens per minute limit"
// +kubebuilder:printcolumn:name="Models",type="string",JSONPath=".spec.models",description="Allowed models for the key"
// +kubebuilder:printcolumn:name="Duration",type="string",JSONPath=".spec.duration",description="Key validity duration"
// +kubebuilder:printcolumn:name="Spend",type="string",JSONPath=".status.spend",description="Current key spend"
// +kubebuilder:printcolumn:name="Created",type="string",JSONPath=".status.createdAt",description="Key creation date"
// +kubebuilder:printcolumn:name="Expires",type="string",JSONPath=".status.expires",description="Key expiration date"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the key"

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
